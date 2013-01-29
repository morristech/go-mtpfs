package mtp

import (
	"testing"
	"bytes"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"
)

func Fatal(e ...interface{}) {
	fmt.Println("fatal:", e)
	os.Exit(1)
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
	var str PropString
	str.Value = "hanwen's nexus 7"
	err = dev.SetDevicePropValue(DPC_MTP_DeviceFriendlyName, &str)
	if err != nil {
		t.Log("SetDevicePropValue failed:", err)
	}

	str = PropString{}
	err = dev.GetDevicePropValue(DPC_MTP_DeviceFriendlyName, &str)
	if err != nil {
		t.Log("GetDevicePropValue failed:", err)
	} else {
		t.Logf("Device Value: %#v\n", str)
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

func TestDevice(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	flag.Parse()

	dev, err := SelectDevice("")
	if err != nil {
		Fatal(err)
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
	sids := StorageIDs{}
	err := dev.GetStorageIDs(&sids)
	if err != nil {
		t.Fatalf("GetStorageIDs failed: %v", err)
	} else {
		t.Logf("%#v\n", sids)
	}

	if len(sids.IDs) == 0 {
		t.Fatalf("No storages")
	}

	id := sids.IDs[0]
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

	backBuf := &bytes.Buffer{}
	err = dev.GetObject(handle, backBuf)
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	} else {
		if bytes.Compare(backBuf.Bytes(), data) != 0 {
			t.Fatalf("back comparison failed.")
		}
	}

	err = dev.DeleteObject(handle)
	if err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}
}

