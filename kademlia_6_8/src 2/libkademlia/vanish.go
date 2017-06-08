package libkademlia

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	mathrand "math/rand"
	"time"
	//"sss"
	"sss"
	
	"fmt"
	"os"
)

type VanishingDataObject struct {
	AccessKey  int64
	Ciphertext []byte
	NumberKeys byte
	Threshold  byte
}

func GenerateRandomCryptoKey() (ret []byte) {
	for i := 0; i < 32; i++ {
		ret = append(ret, uint8(mathrand.Intn(256)))
	}
	return
}

func GenerateRandomAccessKey() (accessKey int64) {
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	accessKey = r.Int63()
	return
}

func CalculateSharedKeyLocations(accessKey int64, count int64, epoch int64) (ids []ID) {
	r := mathrand.New(mathrand.NewSource(accessKey + epoch))
	ids = make([]ID, count)
	for i := int64(0); i < count; i++ {
		for j := 0; j < IDBytes; j++ {
			ids[i][j] = uint8(r.Intn(256))
		}
	}
	return
}

func encrypt(key []byte, text []byte) (ciphertext []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	ciphertext = make([]byte, aes.BlockSize+len(text))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], text)
	return
}

func decrypt(key []byte, ciphertext []byte) (text []byte) {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	if len(ciphertext) < aes.BlockSize {
		panic("ciphertext is not long enough")
	}

	temp_ciphertext := make([]byte, len(ciphertext))
	copy(temp_ciphertext, ciphertext)

	iv := temp_ciphertext[:aes.BlockSize]
	temp_ciphertext = temp_ciphertext[aes.BlockSize:]


	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(temp_ciphertext, temp_ciphertext)
	return temp_ciphertext
}

func (k *Kademlia) VanishData(vdoID ID, data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int, epoch int64) (vdo VanishingDataObject, err error) {
	K := GenerateRandomCryptoKey()
	C := encrypt(K, data)
	keysMap, err := sss.Split(numberKeys, threshold, K)

	if err != nil {
		return VanishingDataObject{}, err
	}

	L := GenerateRandomAccessKey()

	locationsList := CalculateSharedKeyLocations(L, int64(numberKeys), epoch)
	//fmt.Println("lclist",locationsList)
	for index, locationID := range(locationsList) {
		key := uint8(index + 1)
		newKeyValueData := append([]byte{key}, keysMap[key][:]...) // key + value
		_, err = k.DoIterativeStore(locationID, newKeyValueData)
		if err != nil {
			return VanishingDataObject{}, &CommandFailed{"vanishdata store failed"}
		}
	}

	vdo = VanishingDataObject{AccessKey:L, Ciphertext:C, NumberKeys:numberKeys, Threshold:threshold}

	k.VDOmap[vdoID] = vdo

	return
}

func (k *Kademlia) UnvanishCombineKey(vdo VanishingDataObject) (key []byte) {
	L := vdo.AccessKey
	T := vdo.Threshold
	numberKeys := vdo.NumberKeys
	temp_epoch := k.getEpoch()
	locationsList := CalculateSharedKeyLocations(L, int64(numberKeys), temp_epoch)
	splitKeyMap := make(map[byte][]byte)

	fmt.Println("start unvanish")
	for _, locationID := range(locationsList){
		//fmt.Println("len:",len(splitKeyMap))
		if (len(splitKeyMap) >= int(T)) {
			break
		}
		value1,_ := k.LocalFindValue(locationID)
		//fmt.Println("valuevalue1:")
		if value1 == nil {

			value, err := k.DoIterativeFindValue(locationID)
			//fmt.Println("valuevalue2:",value)
			if value != nil {
				//fmt.Println("herereerere:",value[0],value[1:])

				splitKeyMap[byte(value[0])] = value[1:]
			}
			if err != nil {
				fmt.Println(err.Error())

			}
		}else{
			splitKeyMap[byte(value1[0])] = value1[1:]
		}


	}

	if len(splitKeyMap) < int(T) {

		locationsList := CalculateSharedKeyLocations(L, int64(numberKeys), temp_epoch - 1)
		splitKeyMap = make(map[byte][]byte)

		fmt.Println("start unvanish")
		for _, locationID := range(locationsList){
			fmt.Println("len:",len(splitKeyMap))
			if (len(splitKeyMap) >= int(T)) {
				break
			}
			value1,_ := k.LocalFindValue(locationID)
			//fmt.Println("valuevalue111111:")
			if value1 == nil {

				value, err := k.DoIterativeFindValue(locationID)
				//fmt.Println("valuevalue222222:",value)
				if value != nil {
					//fmt.Println("herereerere:",value[0],value[1:])

					splitKeyMap[byte(value[0])] = value[1:]
				}
				if err != nil {
					fmt.Println(err.Error())

				}
			}else{
				splitKeyMap[byte(value1[0])] = value1[1:]
			}

		}
	}
	if len(splitKeyMap) < int(T) {
		fmt.Fprintf(os.Stderr,"Invalid Access Key or cannot find shared keys")
	}else {
		key = sss.Combine(splitKeyMap)
	}
	return
}


func (k *Kademlia) UnvanishData(vdo VanishingDataObject) (data []byte, err error) {
	combinedKey :=k.UnvanishCombineKey(vdo)
	C := vdo.Ciphertext
	data = decrypt(combinedKey, C)
	err = nil
	return
}


// for refreshing VDO splitted keys location (go after vanished the data)
// time is based on UTC time, refreshing happen at refreshEpochOffset seconds before epoch time points
func (k *Kademlia) UpdateTimeOutVdo(timeoutSeconds int) {

	vdo := <- k.VanishTimeOutVDOChan
	startTime := <- k.VanishTimeOutStartChan

	numberKeys := vdo.NumberKeys
	threshold := vdo.Threshold
	L := vdo.AccessKey
	ticker := time.NewTicker(time.Millisecond * 1000)

	for _ = range ticker.C {
		//print(".")
		TimeNow := time.Now().UTC()
		SecondsNowInWeek := int(TimeNow.Weekday())*24*3600 + int(TimeNow.Hour())*3600 +
			int(TimeNow.Minute())*60 + int(TimeNow.Second())
		if (SecondsNowInWeek - startTime >= timeoutSeconds) {
			break
		}
		if ((SecondsNowInWeek + refreshEpochOffset) % epochSeconds == 0 && SecondsNowInWeek - startTime < timeoutSeconds) {
			println("do refreshing VDO keys")
			epoch := (SecondsNowInWeek + refreshEpochOffset) / epochSeconds
			combinedKey := k.UnvanishCombineKey(vdo)
			keysMap, _ := sss.Split(numberKeys, threshold, combinedKey)
			locationsList := CalculateSharedKeyLocations(L, int64(numberKeys), int64(epoch))
			fmt.Println("lclist",locationsList)

			for index, locationID := range(locationsList) {
				key := uint8(index + 1)
				newKeyValueData := append([]byte{key}, keysMap[key][:]...) // key + value
				_, _ = k.DoIterativeStore(locationID, newKeyValueData)
			}
			println("do refreshing done!")
		}
	}
}


func (k *Kademlia) getEpoch() (epoch int64) {
	TimeNow := time.Now().UTC()
	SecondsNowInWeek := int(TimeNow.Weekday())*24*3600 + int(TimeNow.Hour())*3600 +
		int(TimeNow.Minute())*60 + int(TimeNow.Second())
	epoch = int64(SecondsNowInWeek / epochSeconds)
	return
}