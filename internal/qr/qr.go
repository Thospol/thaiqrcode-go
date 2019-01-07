package qr

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"unicode/utf8"

	"github.com/howeyc/crc16"
)

type QR struct {
	PayloadFormatIndicator                              string // 00
	PointOfInitiationMethod                             string // 01; Mandatory
	Merchant                                            QRMerchant
	Transaction                                         QRTransaction
	CountryCode                                         string // 58; Mandatory
	AdditionalData                                      QRAdditionalData
	CRC                                                 string // 63; Mandatory; No Value = Auto-gen
	DataObjectForMerchantAccountInformationByMasterCard string // 51
}

type QRMerchant struct {
	ID           QRMerchantID
	CategoryCode string // 52; Mandatory
	Name         string // 59; Mandatory
	City         string // 60; Mandatory
}

type QRMerchantID struct {
	Visa                 string                           // 02, 03
	MasterCard           string                           // 04, 05
	CUP                  string                           // 04, 05
	JCB                  string                           // n/a
	UnionPay             string                           //15,16
	EMVCo                string                           //17-25
	AMEX                 string                           // n/a
	TPN                  string                           // 26
	PromptCard           string                           // 27
	VisaLocal            string                           // 28
	PromptPay            QRMerchantIDPromptPay            //29
	PromptPayBillPayment QRMerchantIDPromptPayBillPayment //30
	API                  QRMerchantIDPromptPayAPI         // 31
}

type QRMerchantIDPromptPay struct { //29
	AID               string //00
	MobileNumber      string //01
	NationalID        string //02
	EWalletID         string //03
	BankAccount       string //04
	NationalEWalletID string //05
}

type QRMerchantIDPromptPayBillPayment struct { //30
	AID        string //00
	BillerID   string //01
	Reference1 string //02
	Reference2 string //03
}

type QRMerchantIDPromptPayAPI struct { //31
	AID            string //00
	AcquirerID     string //01
	MerchantID     string //02
	TransactionRef string //03
	ReferenceNo    string //04
	TerminalID     string //05
}

type QRTransaction struct {
	CurrencyCode string // 53; Mandatory
	Amount       string // 54
}

type QRAdditionalData struct {
	BillNumber                    string //01
	MobileNumber                  string //02
	StoreID                       string //03
	LoyaltyNumber                 string //04
	ReferenceID                   string //05
	ConsumerID                    string //06
	TerminalID                    string //07
	PurposeOfTransaction          string //08
	AdditionalConsumerDataRequest string //09
}

var stringWithNoCRC bytes.Buffer
var crcFromStr string

func checkCRC(str string, crc string) error {
	validCRC := crc16.ChecksumCCITTFalse([]byte(str)) // Re-generate CRC for checking
	//log.Println("checkCRC: string:: ", str)
	crcInt, _ := strconv.ParseInt("0x"+crc, 0, 64) // Convert CRC(string) to int
	//fmt.Println("crcInt: ", crcInt)
	if validCRC != uint16(crcInt) {
		fmt.Println("String Received: ", str)
		fmt.Printf("ValidCRC: %X\n", validCRC)
		fmt.Println("CRC Received: ", crc)
		return errors.New("Error, CRC is mismatch or QR(string) is invalid.")
	}
	return nil
}

func ConvertStringToMap(s string) (map[string]string, error) {
	// Decode 1st Phase : From string to map
	var m map[string]string
	m = make(map[string]string)
	stringWithNoCRC.Reset() //Reset for sub-tag
	for i := 0; i < utf8.RuneCountInString(s); i++ {
		// Get key
		if i+2 >= utf8.RuneCountInString(s) { // Invalid QR as length is longer than acceptable
			//log.Println("QRVisa(string) is invalid")
			return m, errors.New("QRVisa(string) is invalid")
		}
		//key := s[i : i+2]
		key := string([]rune(s)[i : i+2])
		i = i + 2
		// Get length
		//ls := s[i : i+2]
		ls := string([]rune(s)[i : i+2])

		i = i + 2
		l64, _ := strconv.ParseInt(ls, 10, 0)
		l := int(l64)

		if i+l-1 >= utf8.RuneCountInString(s) { // Invalid QR as length is longer than acceptable
			//log.Println("QRVisa(string) is invalid")
			return m, errors.New("QRVisa(string) is invalid")
		}
		// Get value
		//val := s[i : i+l]
		val := string([]rune(s)[i : i+l])

		i = i + l - 1
		if key == "63" {
			crcFromStr = val
		}
		m[key] = val
		if key != "63" {
			lengthForStringNoCRC := fmt.Sprintf("%02d", l)
			stringWithNoCRC.WriteString(key + lengthForStringNoCRC + val)
		}
	}
	fmt.Println("MapKey:", m)
	return m, nil
}

func ConvertMapToQR(m map[string]string) (*QR, error) {
	// Decode 2nd Phase : From Map to QR struct
	stringToCheckCRC := fmt.Sprintf("%s6304", stringWithNoCRC.String())
	log.Println("String with no crc in decode 2nd step is: ", stringToCheckCRC)
	err := checkCRC(stringToCheckCRC, crcFromStr)
	if err != nil {
		//log.Println(err)
		return nil, err
	}
	// Promptpay
	promptPay := make(map[string]string)
	if m["29"] != "" {
		promptPay, err = ConvertStringToMap(m["29"])
		if err != nil {
			//log.Errorln(err)
			return nil, err
		}
		if promptPay["00"] != "A000000677010111" {
			return nil, errors.New("invalid subtag 'Credit Transfer with PromptPayID'")
		}
	}

	promptPayBillPayment := make(map[string]string)
	if m["30"] != "" {
		promptPayBillPayment, err = ConvertStringToMap(m["30"])
		if err != nil {
			//log.Errorln(err)
			return nil, err
		}
		if promptPayBillPayment["00"] != "A000000677010112" {
			return nil, errors.New("invalid subtag 'Promptpay Bill Payment'")
		}
	}
	api := make(map[string]string)
	if m["31"] != "" {
		api, err = ConvertStringToMap(m["31"])
		if err != nil {
			//log.Errorln(err)
			return nil, err
		}
		//if api["00"] != "A000000677010113" { // remove validation in lib -> move to controllers
		//	return nil, errors.New("invalid subtag 'Credit Transfer with PromptPayID/Promptpay Bill Payment'")
		//}
	}

	additional := make(map[string]string)
	if m["62"] != "" {
		additional, err = ConvertStringToMap(m["62"])
		if err != nil {
			//log.Errorln(err)
			return nil, err
		}
	}

	if m["58"] != "TH" {
		return nil, errors.New("invalid country code, expected 'TH'")
	}

	if m["53"] != "764" {
		return nil, errors.New("invalid transaction currency code, expected '764'")
	}

	qrTnx := QRTransaction{
		CurrencyCode: m["53"],
		Amount:       m["54"],
	}
	qrMerPP := QRMerchantIDPromptPay{
		AID:               promptPay["00"],
		MobileNumber:      promptPay["01"],
		NationalID:        promptPay["02"],
		EWalletID:         promptPay["03"],
		BankAccount:       promptPay["04"],
		NationalEWalletID: promptPay["05"],
	}
	qrMerPPBillPay := QRMerchantIDPromptPayBillPayment{
		AID:        promptPayBillPayment["00"],
		BillerID:   promptPayBillPayment["01"],
		Reference1: promptPayBillPayment["02"],
		Reference2: promptPayBillPayment["03"],
	}
	qrMerAPI := QRMerchantIDPromptPayAPI{
		AID:            api["00"],
		AcquirerID:     api["01"],
		MerchantID:     api["02"],
		TransactionRef: api["03"],
		ReferenceNo:    api["04"],
		TerminalID:     api["05"],
	}
	qrMerID := QRMerchantID{
		Visa:                 m["02"],
		MasterCard:           m["04"],
		CUP:                  m["14"],
		TPN:                  m["26"],
		PromptCard:           m["27"],
		VisaLocal:            m["28"],
		UnionPay:             m["15"],
		EMVCo:                m["17"],
		PromptPay:            qrMerPP,
		PromptPayBillPayment: qrMerPPBillPay,
		API:                  qrMerAPI,
	}
	qrMer := QRMerchant{
		ID:           qrMerID,
		CategoryCode: m["52"],
		Name:         m["59"],
		City:         m["60"],
	}
	qrAdditionalData := QRAdditionalData{
		BillNumber:                    additional["01"],
		MobileNumber:                  additional["02"],
		StoreID:                       additional["03"],
		LoyaltyNumber:                 additional["04"],
		ReferenceID:                   additional["05"],
		ConsumerID:                    additional["06"],
		TerminalID:                    additional["07"],
		PurposeOfTransaction:          additional["08"],
		AdditionalConsumerDataRequest: additional["09"],
	}
	qr := QR{
		PayloadFormatIndicator:  m["00"],
		PointOfInitiationMethod: m["01"],
		Merchant:                qrMer,
		AdditionalData:          qrAdditionalData,
		CountryCode:             m["58"],
		CRC:                     m["63"],
		Transaction:             qrTnx,
		DataObjectForMerchantAccountInformationByMasterCard: m["51"],
	}

	// Length Check
	if qr.PayloadFormatIndicator != "" && utf8.RuneCountInString(qr.PayloadFormatIndicator) != 2 {
		//log.Errorln("Invalid QR (utf8.RuneCountInString(qr.PayloadFormatIndicator) must be 2 but got ", utf8.RuneCountInString(qr.PayloadFormatIndicator), ")")
		return nil, fmt.Errorf("Invalid QR (utf8.RuneCountInString(qr.PayloadFormatIndicator) must be 2 but got %d", utf8.RuneCountInString(qr.PayloadFormatIndicator))
	}

	if qr.PointOfInitiationMethod != "" && utf8.RuneCountInString(qr.PointOfInitiationMethod) != 2 {
		//log.Errorln("Invalid QR (utf8.RuneCountInString(qr.PointOfInitiationMethod) must be 2 but got ", utf8.RuneCountInString(qr.PointOfInitiationMethod), ")")
		return nil, fmt.Errorf("Invalid QR (utf8.RuneCountInString(qr.PointOfInitiationMethod) must be 2 but got %d", utf8.RuneCountInString(qr.PointOfInitiationMethod))
	}

	if qr.Merchant.CategoryCode != "" && utf8.RuneCountInString(qr.Merchant.CategoryCode) != 4 {
		//log.Errorln("Invalid QR (utf8.RuneCountInString(qr.Merchant.CategoryCode) must be 4 but got ", utf8.RuneCountInString(qr.Merchant.CategoryCode), ")")
		return nil, fmt.Errorf("Invalid QR (utf8.RuneCountInString(qr.Merchant.CategoryCode) must be 4 but got %d", utf8.RuneCountInString(qr.Merchant.CategoryCode))
	}

	if qr.Transaction.CurrencyCode != "" && utf8.RuneCountInString(qr.Transaction.CurrencyCode) != 3 {
		//log.Errorln("Invalid QR (utf8.RuneCountInString(qr.Transaction.CurrencyCode) must be 3 but got ", utf8.RuneCountInString(qr.Transaction.CurrencyCode), ")")
		return nil, fmt.Errorf("Invalid QR (utf8.RuneCountInString(qr.Transaction.CurrencyCode) must be 3 but got %d", utf8.RuneCountInString(qr.Transaction.CurrencyCode))
	}

	if qr.CountryCode != "" && utf8.RuneCountInString(qr.CountryCode) != 2 {
		//log.Errorln("Invalid QR (utf8.RuneCountInString(qr.CountryCode) must be 2 but got ", utf8.RuneCountInString(qr.CountryCode), ")")
		return nil, fmt.Errorf("Invalid QR (utf8.RuneCountInString(qr.CountryCode) must be 2 but got %d", utf8.RuneCountInString(qr.CountryCode))
	}

	if qr.CRC != "" && utf8.RuneCountInString(qr.CRC) != 4 {
		//log.Errorln("Invalid QR (utf8.RuneCountInString(qr.CRC) must be 4 but got ", utf8.RuneCountInString(qr.CRC), ")")
		return nil, fmt.Errorf("Invalid QR (utf8.RuneCountInString(qr.CRC) must be 4 but got %d", utf8.RuneCountInString(qr.CRC))
	}

	if qr.DataObjectForMerchantAccountInformationByMasterCard != "" && utf8.RuneCountInString(qr.DataObjectForMerchantAccountInformationByMasterCard) != 25 {
		return nil, fmt.Errorf("Invalid QR (utf8.RuneCountInString(qr.DataObjectForMerchantAccountInformationByMasterCard) must be 25 but got %d", utf8.RuneCountInString(qr.DataObjectForMerchantAccountInformationByMasterCard))
	}

	return &qr, nil
}

func DecodeQRVisa(s string) (*QR, error) {
	stringConverted, err := ConvertStringToMap(s)
	if err != nil {
		//log.Errorln(err)
		return new(QR), err //If error; return empty QR struct
	}

	mapConverted, err := ConvertMapToQR(stringConverted)
	if err != nil {
		return new(QR), err
	}
	return mapConverted, nil
}

func ConvertQRToMap(qr *QR) (map[string]string, error) {
	// Encode 1st Phase : From string to map
	m := map[string]string{}

	// Length Check

	if qr.PayloadFormatIndicator != "" && utf8.RuneCountInString(qr.PayloadFormatIndicator) != 2 {
		//log.Errorln("utf8.RuneCountInString(qr.PayloadFormatIndicator) must be 2 (got ", utf8.RuneCountInString(qr.PayloadFormatIndicator), ")")
		return nil, fmt.Errorf("utf8.RuneCountInString(qr.PayloadFormatIndicator) must be 2 (got %d)", utf8.RuneCountInString(qr.PayloadFormatIndicator))
	}

	if qr.PointOfInitiationMethod != "" && utf8.RuneCountInString(qr.PointOfInitiationMethod) != 2 {
		//log.Errorln("utf8.RuneCountInString(qr.PointOfInitiationMethod) must be 2 (got ", utf8.RuneCountInString(qr.PointOfInitiationMethod), ")")
		return nil, fmt.Errorf("utf8.RuneCountInString(qr.PointOfInitiationMethod) must be 2 (got %d)", utf8.RuneCountInString(qr.PointOfInitiationMethod))
	}

	if qr.Merchant.CategoryCode != "" && utf8.RuneCountInString(qr.Merchant.CategoryCode) != 4 {
		//log.Errorln("utf8.RuneCountInString(qr.Merchant.CategoryCode) must be 4 (got ", utf8.RuneCountInString(qr.Merchant.CategoryCode), ")")
		return nil, fmt.Errorf("utf8.RuneCountInString(qr.Merchant.CategoryCode) must be 4 (got %d)", utf8.RuneCountInString(qr.Merchant.CategoryCode))
	}

	if qr.Transaction.CurrencyCode != "" && utf8.RuneCountInString(qr.Transaction.CurrencyCode) != 3 {
		//log.Errorln("utf8.RuneCountInString(qr.Transaction.CurrencyCode) must be 3 (got ", utf8.RuneCountInString(qr.Transaction.CurrencyCode), ")")
		return nil, fmt.Errorf("utf8.RuneCountInString(qr.Transaction.CurrencyCode) must be 3 (got %d)", utf8.RuneCountInString(qr.Transaction.CurrencyCode))
	}

	if qr.CountryCode != "" && utf8.RuneCountInString(qr.CountryCode) != 2 {
		//log.Errorln("utf8.RuneCountInString(qr.CountryCode) must be 2 (got ", utf8.RuneCountInString(qr.CountryCode), ")")
		return nil, fmt.Errorf("utf8.RuneCountInString(qr.CountryCode) must be 2 (got %d)", utf8.RuneCountInString(qr.CountryCode))
	}

	if qr.CRC != "" && utf8.RuneCountInString(qr.CRC) != 4 {
		//log.Errorln("utf8.RuneCountInString(qr.CRC) must be 4 (got ", utf8.RuneCountInString(qr.CRC), ")")
		return nil, fmt.Errorf("utf8.RuneCountInString(qr.CRC) must be 4 (got %d)", utf8.RuneCountInString(qr.CRC))
	}

	if qr.DataObjectForMerchantAccountInformationByMasterCard != "" && utf8.RuneCountInString(qr.DataObjectForMerchantAccountInformationByMasterCard) != 4 {
		//log.Errorln("utf8.RuneCountInString(qr.DataObjectForMerchantAccountInformationByMasterCard) must be 25 (got ", utf8.RuneCountInString(qr.DataObjectForMerchantAccountInformationByMasterCard), ")")
		return nil, fmt.Errorf("utf8.RuneCountInString(qr.DataObjectForMerchantAccountInformationByMasterCard) must be 25 (got %d)", utf8.RuneCountInString(qr.DataObjectForMerchantAccountInformationByMasterCard))
	}

	m["00"] = qr.PayloadFormatIndicator
	m["01"] = qr.PointOfInitiationMethod

	m["02"] = qr.Merchant.ID.Visa
	m["04"] = qr.Merchant.ID.MasterCard
	m["14"] = qr.Merchant.ID.CUP
	m["15"] = qr.Merchant.ID.UnionPay
	m["17"] = qr.Merchant.ID.EMVCo
	m["26"] = qr.Merchant.ID.TPN
	m["27"] = qr.Merchant.ID.PromptCard
	m["28"] = qr.Merchant.ID.VisaLocal

	// if field 29 has sub-field
	if qr.Merchant.ID.PromptPay.AID != "" || qr.Merchant.ID.PromptPay.MobileNumber != "" || qr.Merchant.ID.PromptPay.NationalID != "" || qr.Merchant.ID.PromptPay.EWalletID != "" || qr.Merchant.ID.PromptPay.BankAccount != "" || qr.Merchant.ID.PromptPay.NationalEWalletID != "" {
		var str bytes.Buffer

		if qr.Merchant.ID.PromptPay.AID != "" {
			str.WriteString("00")
			if utf8.RuneCountInString(qr.Merchant.ID.PromptPay.AID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.PromptPay.AID))) // Write length
			str.WriteString(qr.Merchant.ID.PromptPay.AID)
		}

		if qr.Merchant.ID.PromptPay.MobileNumber != "" {
			str.WriteString("01")
			if utf8.RuneCountInString(qr.Merchant.ID.PromptPay.MobileNumber) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.PromptPay.MobileNumber))) // Write length
			str.WriteString(qr.Merchant.ID.PromptPay.MobileNumber)
		}

		if qr.Merchant.ID.PromptPay.NationalID != "" {
			str.WriteString("02")
			if utf8.RuneCountInString(qr.Merchant.ID.PromptPay.NationalID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.PromptPay.NationalID))) // Write length
			str.WriteString(qr.Merchant.ID.PromptPay.NationalID)
		}

		if qr.Merchant.ID.PromptPay.EWalletID != "" {
			str.WriteString("03")
			if utf8.RuneCountInString(qr.Merchant.ID.PromptPay.EWalletID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.PromptPay.EWalletID))) // Write length
			str.WriteString(qr.Merchant.ID.PromptPay.EWalletID)
		}

		if qr.Merchant.ID.PromptPay.BankAccount != "" {
			str.WriteString("04")
			if utf8.RuneCountInString(qr.Merchant.ID.PromptPay.BankAccount) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.PromptPay.BankAccount))) // Write length
			str.WriteString(qr.Merchant.ID.PromptPay.BankAccount)
		}

		if qr.Merchant.ID.PromptPay.NationalEWalletID != "" {
			str.WriteString("05")
			if utf8.RuneCountInString(qr.Merchant.ID.PromptPay.NationalEWalletID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.PromptPay.NationalEWalletID))) // Write length
			str.WriteString(qr.Merchant.ID.PromptPay.NationalEWalletID)
		}

		m["29"] = str.String() // Write sub-data to m["29"]
	}

	// if field 30 has sub-field
	if qr.Merchant.ID.PromptPayBillPayment.AID != "" || qr.Merchant.ID.PromptPayBillPayment.BillerID != "" || qr.Merchant.ID.PromptPayBillPayment.Reference1 != "" || qr.Merchant.ID.PromptPayBillPayment.Reference2 != "" {

		var str bytes.Buffer

		if qr.Merchant.ID.PromptPayBillPayment.AID != "" {
			str.WriteString("00")
			if utf8.RuneCountInString(qr.Merchant.ID.PromptPayBillPayment.AID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.PromptPayBillPayment.AID))) // Write length
			str.WriteString(qr.Merchant.ID.PromptPayBillPayment.AID)
		}

		if qr.Merchant.ID.PromptPayBillPayment.BillerID != "" {
			str.WriteString("01")
			if utf8.RuneCountInString(qr.Merchant.ID.PromptPayBillPayment.BillerID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.PromptPayBillPayment.BillerID))) // Write length
			str.WriteString(qr.Merchant.ID.PromptPayBillPayment.BillerID)
		}

		if qr.Merchant.ID.PromptPayBillPayment.Reference1 != "" {
			str.WriteString("02")
			if utf8.RuneCountInString(qr.Merchant.ID.PromptPayBillPayment.Reference1) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.PromptPayBillPayment.Reference1))) // Write length
			str.WriteString(qr.Merchant.ID.PromptPayBillPayment.Reference1)
		}

		if qr.Merchant.ID.PromptPayBillPayment.Reference2 != "" {
			str.WriteString("03")
			if utf8.RuneCountInString(qr.Merchant.ID.PromptPayBillPayment.Reference2) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.PromptPayBillPayment.Reference2))) // Write length
			str.WriteString(qr.Merchant.ID.PromptPayBillPayment.Reference2)
		}

		m["30"] = str.String() // Write sub-data to m["30"]
	}

	// if field 31 has sub-field
	if qr.Merchant.ID.API.AID != "" || qr.Merchant.ID.API.AcquirerID != "" || qr.Merchant.ID.API.MerchantID != "" {

		var str bytes.Buffer

		if qr.Merchant.ID.API.AID != "" {
			str.WriteString("00")
			if utf8.RuneCountInString(qr.Merchant.ID.API.AID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.API.AID))) // Write length
			str.WriteString(qr.Merchant.ID.API.AID)
		}

		if qr.Merchant.ID.API.AcquirerID != "" {
			str.WriteString("01")
			if utf8.RuneCountInString(qr.Merchant.ID.API.AcquirerID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.API.AcquirerID))) // Write length
			str.WriteString(qr.Merchant.ID.API.AcquirerID)
		}

		if qr.Merchant.ID.API.MerchantID != "" {
			str.WriteString("02")
			if utf8.RuneCountInString(qr.Merchant.ID.API.MerchantID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.API.MerchantID))) // Write length
			str.WriteString(qr.Merchant.ID.API.MerchantID)
		}

		if qr.Merchant.ID.API.TransactionRef != "" {
			str.WriteString("03")
			if utf8.RuneCountInString(qr.Merchant.ID.API.TransactionRef) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.API.TransactionRef))) // Write length
			str.WriteString(qr.Merchant.ID.API.TransactionRef)
		}

		if qr.Merchant.ID.API.ReferenceNo != "" {
			str.WriteString("04")
			if utf8.RuneCountInString(qr.Merchant.ID.API.ReferenceNo) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.API.ReferenceNo))) // Write length
			str.WriteString(qr.Merchant.ID.API.ReferenceNo)
		}

		if qr.Merchant.ID.API.TerminalID != "" {
			str.WriteString("05")
			if utf8.RuneCountInString(qr.Merchant.ID.API.TerminalID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.Merchant.ID.API.TerminalID))) // Write length
			str.WriteString(qr.Merchant.ID.API.TerminalID)
		}

		m["31"] = str.String() // Write sub-data to m["31"]
	}

	m["52"] = qr.Merchant.CategoryCode
	m["53"] = qr.Transaction.CurrencyCode
	m["54"] = qr.Transaction.Amount
	m["58"] = qr.CountryCode
	m["59"] = qr.Merchant.Name
	m["60"] = qr.Merchant.City

	// if field 62 has sub-field
	if qr.AdditionalData.BillNumber != "" || qr.AdditionalData.MobileNumber != "" || qr.AdditionalData.StoreID != "" || qr.AdditionalData.LoyaltyNumber != "" || qr.AdditionalData.ReferenceID != "" || qr.AdditionalData.ConsumerID != "" || qr.AdditionalData.TerminalID != "" || qr.AdditionalData.PurposeOfTransaction != "" || qr.AdditionalData.AdditionalConsumerDataRequest != "" {
		var str bytes.Buffer

		if qr.AdditionalData.BillNumber != "" {
			str.WriteString("01")
			if utf8.RuneCountInString(qr.AdditionalData.BillNumber) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.AdditionalData.BillNumber))) // Write length
			str.WriteString(qr.AdditionalData.BillNumber)
		}

		if qr.AdditionalData.MobileNumber != "" {
			str.WriteString("02")
			if utf8.RuneCountInString(qr.AdditionalData.MobileNumber) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.AdditionalData.MobileNumber))) // Write length
			str.WriteString(qr.AdditionalData.MobileNumber)
		}

		if qr.AdditionalData.StoreID != "" {
			str.WriteString("03")
			if utf8.RuneCountInString(qr.AdditionalData.StoreID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.AdditionalData.StoreID))) // Write length
			str.WriteString(qr.AdditionalData.StoreID)
		}

		if qr.AdditionalData.LoyaltyNumber != "" {
			str.WriteString("04")
			if utf8.RuneCountInString(qr.AdditionalData.LoyaltyNumber) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.AdditionalData.LoyaltyNumber))) // Write length
			str.WriteString(qr.AdditionalData.LoyaltyNumber)
		}

		if qr.AdditionalData.ReferenceID != "" {
			str.WriteString("05")
			if utf8.RuneCountInString(qr.AdditionalData.ReferenceID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.AdditionalData.ReferenceID))) // Write length
			str.WriteString(qr.AdditionalData.ReferenceID)
		}

		if qr.AdditionalData.ConsumerID != "" {
			str.WriteString("06")
			if utf8.RuneCountInString(qr.AdditionalData.ConsumerID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.AdditionalData.ConsumerID))) // Write length
			str.WriteString(qr.AdditionalData.ConsumerID)
		}

		if qr.AdditionalData.TerminalID != "" {
			str.WriteString("07")
			if utf8.RuneCountInString(qr.AdditionalData.TerminalID) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.AdditionalData.TerminalID))) // Write length
			str.WriteString(qr.AdditionalData.TerminalID)
		}

		if qr.AdditionalData.PurposeOfTransaction != "" {
			str.WriteString("08")
			if utf8.RuneCountInString(qr.AdditionalData.PurposeOfTransaction) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.AdditionalData.PurposeOfTransaction))) // Write length
			str.WriteString(qr.AdditionalData.PurposeOfTransaction)
		}

		if qr.AdditionalData.AdditionalConsumerDataRequest != "" {
			str.WriteString("09")
			if utf8.RuneCountInString(qr.AdditionalData.AdditionalConsumerDataRequest) < 10 { // Check length if < 10 then write 0 to make 2 digits
				str.WriteString("0")
			}
			str.WriteString(strconv.Itoa(utf8.RuneCountInString(qr.AdditionalData.AdditionalConsumerDataRequest))) // Write length
			str.WriteString(qr.AdditionalData.AdditionalConsumerDataRequest)
		}

		m["62"] = str.String() // Write sub-data to m["62"]
	}

	m["63"] = qr.CRC
	m["63"] = qr.DataObjectForMerchantAccountInformationByMasterCard

	// Length check, each tag must not be longer than 99
	for key, subtag := range m {
		ls := utf8.RuneCountInString(subtag)
		if ls > 99 {
			return nil, fmt.Errorf("length of each tag must not longer than 99, found tag %s with length %d", key, ls)
		}
	}
	return m, nil
}

func ConvertMapToString(mapStr map[string]string) (string, error) {
	// Encode 2nd Phase : Map to string
	var str bytes.Buffer

	// Sort map key
	sortedKey := make([]string, len(mapStr))
	i := 0
	for k := range mapStr {
		sortedKey[i] = k
		i++
	}
	sort.Strings(sortedKey)
	fmt.Println("sortedKeyFrommapStr:", sortedKey)

	for _, keyMap := range sortedKey {
		mapToPrint := mapStr[keyMap]
		if mapToPrint != "" {
			//fmt.Println("Keymap: ", keyMap, ", Value: ", mapToPrint)

			if keyMap == "63" { // If string does have CRC then check if it is correct
				stringToGenCRC := fmt.Sprintf("%s6304", str.String())
				crcGeneratedInt := crc16.ChecksumCCITTFalse([]byte(stringToGenCRC))
				crcGeneratedHex := fmt.Sprintf("%X", crcGeneratedInt)
				if mapToPrint != crcGeneratedHex {
					//log.Errorf("Invalid CRC ! expected %s, but got, %s", crcGeneratedHex, mapStr["63"])
					return "", fmt.Errorf("Invalid CRC ! expected %s, but got, %s", crcGeneratedHex, mapStr["63"])
				}
			}
			str.WriteString(keyMap)
			// Get length of the value and then get length of the length
			lengthOfValue := strconv.Itoa(utf8.RuneCountInString(mapToPrint))
			lengthToWrite := len(lengthOfValue)
			if lengthToWrite == 1 {
				str.WriteString("0") // Add 0 if length has only 1 digit
			}
			str.WriteString(lengthOfValue)
			str.WriteString(mapToPrint)
		}
	}

	if mapStr["63"] == "" { // If string doesn't have CRC then generate and write to string

		//log.Println("String without CRC is : ", str.String())
		stringToGenCRC := fmt.Sprintf("%s6304", str.String())
		crcGeneratedInt := crc16.ChecksumCCITTFalse([]byte(stringToGenCRC))
		crcGeneratedHex := fmt.Sprintf("%X", crcGeneratedInt)

		//log.Println("Generated CRC is : ", crcGeneratedInt)

		str.WriteString("63")
		str.WriteString("04")
		for i := utf8.RuneCountInString(crcGeneratedHex); i < 4; i++ {
			str.WriteString("0") // If length of Hex less than 4 then add 0 before write
		}
		str.WriteString(crcGeneratedHex)
	}

	return str.String(), nil
}
