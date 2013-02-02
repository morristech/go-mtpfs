package mtp

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestAndroid(t *testing.T) {
	dev, err := SelectDevice("")
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	info := DeviceInfo{}
	err = dev.GetDeviceInfo(&info)
	if err != nil {
		t.Log("GetDeviceInfo failed:", err)
	}

	if !strings.Contains(info.MTPExtension, "android.com:") {
		t.Log("no android extensions", info.MTPExtension)
		return
	}

	err = dev.Claim()
	if err != nil {
		t.Log("device claim failed:", err)
	}

	if err = dev.OpenSession(); err != nil {
		t.Fatal("OpenSession failed:", err)
	}

	sids := Uint32Array{}
	err = dev.GetStorageIDs(&sids)
	if err != nil {
		t.Fatalf("GetStorageIDs failed: %v", err)
	}

	if len(sids.Values) == 0 {
		t.Fatalf("No storages")
	}

	id := sids.Values[0]

	// 500 + 512 triggers the null read case on both sides.
	const testSize = 500 + 512
	name := fmt.Sprintf("mtp-doodle-test%x", rand.Int31())

	send := ObjectInfo{
		StorageID:        id,
		ObjectFormat:     OFC_Undefined,
		ParentObject:     0xFFFFFFFF,
		Filename:         name,
		CompressedSize:   uint32(testSize),
		ModificationDate: time.Now(),
		Keywords:         "bla",
	}
	data := make([]byte, testSize)
	for i := range data {
		data[i] = byte('0' + i%10)
	}

	_, _, handle, err := dev.SendObjectInfo(id, 0xFFFFFFFF, &send)
	if err != nil {
		t.Fatal("SendObjectInfo failed:", err)
	} else {
		buf := bytes.NewBuffer(data)
		t.Logf("Sent objectinfo handle: 0x%x\n", handle)
		err = dev.SendObject(buf, int64(len(data)))
		if err != nil {
			t.Log("SendObject failed:", err)
		}
	}

	magicStr := "life universe"
	magicOff := 21
	magicSize := 42

	err = dev.AndroidBeginEditObject(handle)
	if err != nil {
		t.Errorf("AndroidBeginEditObject: %v", err)
		return
	} else {
		err = dev.AndroidTruncate(handle, int64(magicSize))
		if err != nil {
			t.Errorf("AndroidTruncate: %v", err)
		}
		buf := bytes.NewBufferString(magicStr)
		err = dev.AndroidSendPartialObject(handle, int64(magicOff), uint32(buf.Len()), buf)
		if err != nil {
			t.Errorf("AndroidSendPartialObject: %v", err)
		}
		if buf.Len() > 0 {
			t.Errorf("buffer not consumed")
		}
		err = dev.AndroidEndEditObject(handle)
		if err != nil {
			t.Errorf("AndroidEndEditObject: %v", err)
		}
	}

	buf := &bytes.Buffer{}
	err = dev.GetObject(handle, buf)
	if err != nil {
		t.Errorf("GetObject: %v", err)
	}

	if buf.Len() != magicSize {
		t.Errorf("truncate failed:: %v", err)
	}
	for i := 0; i < len(magicStr); i++ {
		data[i+magicOff] = magicStr[i]
	}
	want := string(data[:magicSize])
	if buf.String() != want {
		t.Errorf("read result was %q, want %q", buf.String(), want)
	}

	buf = &bytes.Buffer{}
	err = dev.AndroidGetPartialObject64(handle, buf, int64(magicOff), 5)
	if err != nil {
		t.Errorf("AndroidGetPartialObject64: %v", err)
	}
	want = magicStr[:5]
	got := buf.String()
	if got != want {
		t.Errorf("AndroidGetPartialObject64: got %q want %q", got, want)
	}

	// Try write at end of file.
	err = dev.AndroidBeginEditObject(handle)
	if err != nil {
		t.Errorf("AndroidBeginEditObject: %v", err)
		return
	} else {
		buf := bytes.NewBufferString(magicStr)
		err = dev.AndroidSendPartialObject(handle, int64(magicSize), uint32(buf.Len()), buf)
		if err != nil {
			t.Errorf("AndroidSendPartialObject: %v", err)
		}
		if buf.Len() > 0 {
			t.Errorf("buffer not consumed")
		}
		err = dev.AndroidEndEditObject(handle)
		if err != nil {
			t.Errorf("AndroidEndEditObject: %v", err)
		}
	}
	buf = &bytes.Buffer{}
	err = dev.GetObject(handle, buf)
	if err != nil {
		t.Errorf("GetObject: %v", err)
	}
	want = string(data[:magicSize]) + magicStr
	got = buf.String()
	if got != want {
		t.Errorf("GetObject: got %q want %q", got, want)
	}

	err = dev.DeleteObject(handle)
	if err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}
}

func testDeviceProperties(dev *Device, t *testing.T) {
	var friendly DevicePropDesc
	err := dev.GetDevicePropDesc(DPC_MTP_DeviceFriendlyName, &friendly)
	if err != nil {
		t.Log("friendly failed:", err)
	} else {
		t.Logf("%#v\n", friendly)
	}

	t.Log("setpropt")
	var str StringValue
	str.Value = "hanwen's nexus 7"
	err = dev.SetDevicePropValue(DPC_MTP_DeviceFriendlyName, &str)
	if err != nil {
		t.Log("SetDevicePropValue failed:", err)
	}

	str = StringValue{}
	err = dev.GetDevicePropValue(DPC_MTP_DeviceFriendlyName, &str)
	if err != nil {
		t.Log("GetDevicePropValue failed:", err)
	} else {
		t.Logf("Device Value: %#v\n", str)
	}

	props := Uint16Array{}
	err = dev.GetObjectPropsSupported(OFC_Undefined, &props)
	if err != nil {
		t.Logf("GetObjectPropsSupported failed: %v\n", err)
	} else {
		t.Logf("GetObjectPropsSupported (OFC_Undefined) value: %s\n", getNames(OPC_names, props.Values))
	}

	for _, p := range props.Values {
		var objPropDesc ObjectPropDesc
		if p == OPC_PersistantUniqueObjectIdentifier {
			// can't deal with int128.
			continue
		}

		err = dev.GetObjectPropDesc(p, OFC_Undefined, &objPropDesc)
		name := OPC_names[int(p)]
		if err != nil {
			t.Logf("GetObjectPropDesc(%s) failed: %v\n", name, err)
		} else {
			t.Logf("GetObjectPropDesc(%s) value: %#v %T\n", name, objPropDesc,
				InstantiateType(objPropDesc.DataType).Interface())
		}
	}

	err = dev.ResetDeviceProp(DPC_MTP_DeviceFriendlyName)
	if err != nil {
		t.Log("ResetDeviceProp:", err)
	}

	var battery DevicePropDesc
	err = dev.GetDevicePropDesc(DPC_BatteryLevel, &friendly)
	if err != nil {
		t.Log("battery failed:", err)
	} else {
		t.Logf("%#v\n", battery)
	}
}
func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestDevice(t *testing.T) {
	dev, err := SelectDevice("")
	if err != nil {
		t.Fatal(err)
	}
	defer dev.Close()

	i, _ := dev.Id()
	t.Log("device:", i)
	err = dev.Claim()
	if err != nil {
		t.Log("device claim failed:", err)
	}

	//t.Log(dev.OpenSession())
	t.Log("devinfo:")
	//dev.DebugPrint = true
	info := DeviceInfo{}
	err = dev.GetDeviceInfo(&info)
	if err != nil {
		t.Log("GetDeviceInfo failed:", err)
	} else {
		t.Logf("%v\n", &info)
	}

	err = dev.OpenSession()
	if err != nil {
		t.Log("Session failed:", err)
	}

	testDeviceProperties(dev, t)
	if err != nil {
		t.Error(err)
	}
	testStorage(dev, t)
	if err != nil {
		t.Error(err)
	}
}

func testStorage(dev *Device, t *testing.T) {
	sids := Uint32Array{}
	err := dev.GetStorageIDs(&sids)
	if err != nil {
		t.Fatalf("GetStorageIDs failed: %v", err)
	} else {
		t.Logf("%#v\n", sids)
	}

	if len(sids.Values) == 0 {
		t.Fatalf("No storages")
	}

	id := sids.Values[0]
	var storageInfo StorageInfo
	dev.GetStorageInfo(id, &storageInfo)
	if err != nil {
		t.Fatalf("GetStorageInfo failed:", err)
	} else {
		t.Logf("%#v\n", storageInfo)
	}

	resp, err := dev.GenericRPC(OC_GetNumObjects, []uint32{id, 0x0, 0x0})
	if err != nil {
		t.Fatalf("GenericRPC failed:", err)
	} else {
		t.Logf("num objects %#v\n", resp)
	}

	// 500 + 512 triggers the null read case on both sides.
	const testSize = 500 + 512
	data := make([]byte, testSize)
	for i := range data {
		data[i] = byte(i % 256)
	}

	name := fmt.Sprintf("mtp-doodle-test%x", rand.Int31())
	buf := bytes.NewBuffer(data)
	send := ObjectInfo{
		StorageID:        id,
		ObjectFormat:     OFC_Undefined,
		ParentObject:     0xFFFFFFFF,
		Filename:         name,
		CompressedSize:   uint32(len(data)),
		ModificationDate: time.Now(),
		Keywords:         "bla",
	}

	_, _, handle, err := dev.SendObjectInfo(id, 0xFFFFFFFF, &send)
	if err != nil {
		t.Fatalf("SendObjectInfo failed:", err)
	} else {
		t.Logf("Sent objectinfo handle: 0x%x\n", handle)
		err = dev.SendObject(buf, int64(len(data)))
		if err != nil {
			t.Log("SendObject failed:", err)
		}
	}

	hs := ObjectHandles{}
	err = dev.GetObjectHandles(id,
		OFC_Undefined,
		//OFC_Association,
		0xFFFFFFFF, &hs)

	if err != nil {
		t.Fatalf("GetObjectHandles failed: %v", err)
	} else {
		found := false
		for _, h := range hs.Handles {
			if h == handle {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("uploaded file is not there")
		}
	}

	var backInfo ObjectInfo
	err = dev.GetObjectInfo(handle, &backInfo)
	if err != nil {
		t.Fatalf("GetObjectInfo failed: %v", err)
	} else {
		t.Logf("info %#v\n", backInfo)
	}

	var objSize Uint64Value
	err = dev.GetObjectPropValue(handle, OPC_ObjectSize, &objSize)
	if err != nil {
		t.Fatalf("GetObjectPropValue failed: %v", err)
	} else {
		t.Logf("info %#v\n", objSize)
	}

	if objSize.Value != testSize {
		t.Errorf("object size error: got %v, want %v", objSize.Value, testSize)
	}

	backBuf := &bytes.Buffer{}
	err = dev.GetObject(handle, backBuf)
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	} else {
		if bytes.Compare(backBuf.Bytes(), data) != 0 {
			t.Fatalf("back comparison failed.")
		}
	}

	newName := fmt.Sprintf("mtp-doodle-test%x", rand.Int31())
	err = dev.SetObjectPropValue(handle, OPC_ObjectFileName, &StringValue{newName})
	if err != nil {
		t.Errorf("error renaming object: %v", err)
	}

	err = dev.DeleteObject(handle)
	if err != nil {
		t.Fatalf("DeleteObject failed: %v", err)

	}
}
