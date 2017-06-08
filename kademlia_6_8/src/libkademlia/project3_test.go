package libkademlia


import (
	"testing"
	"strconv"
	"fmt"
	"bytes"
	//"time"
	//"time"
)

// Normal Case


func TestVanishData2(t *testing.T) {
	var id ID
	instance2 := NewKademliaWithId("localhost:7100",id)
	data :=[]byte("Distributed System")
	numberKeys := byte(20)
	threshold := byte(10)
	timeoutSeconds := 100



	//generate 60 nodes
	// Generate 60 nodes
	node_set := make([][]*Kademlia, 3)
	for i := 0; i < 3; i++ {
		node_set[i] = make([]*Kademlia, 20)
		for j := 0; j < 20; j++ {
			address := "localhost:" + strconv.Itoa(7100+ i*20+(j+1))
			tempID := instance2.SelfContact.NodeID
			tempID[19] = uint8(32 + i*20 + j + 1)
			node_set[i][j] = NewKademliaWithId(address, tempID)
		}
	}

	//initialize instance2's k-bucket
	// Initialize node1's k-bucket
	host, port, _ := StringToIpPort("localhost:7101")
	instance2.DoPing(host, port)
	host, port, _ = StringToIpPort("localhost:7121")
	instance2.DoPing(host, port)
	host, port, _ = StringToIpPort("localhost:7141")
	instance2.DoPing(host, port)

	// Initialize three points' k bucket.
	for i := 0; i < 3; i++ {
		for j := 0; j < 20; j++ {
			address := "localhost:" + strconv.Itoa(7100+ i*20+(j+1))
			host, port, _ = StringToIpPort(address)
			node_set[i][0].DoPing(host, port)
		}
	}

	new_vdoID := NewRandomID()
	vdo := instance2.Vanish(new_vdoID,data,numberKeys,threshold,timeoutSeconds)
	if instance2.VDOmap[new_vdoID].AccessKey !=vdo.AccessKey {
		t.Error("The vanish object have not been stored in the DHT map")
	}
}


func TestVanishData3(t *testing.T) {
	var id ID
	instance3 := NewKademliaWithId("localhost:8100",id)
	data :=[]byte("Distributed System")
	numberKeys := byte(20)
	threshold := byte(10)
	timeoutSeconds := 0

	//generate 60 nodes
	// Generate 60 nodes
	node_set := make([][]*Kademlia, 3)
	for i := 0; i < 3; i++ {
		node_set[i] = make([]*Kademlia, 20)
		for j := 0; j < 20; j++ {
			address := "localhost:" + strconv.Itoa(8100+ i*20+(j+1))
			tempID := instance3.SelfContact.NodeID
			tempID[19] = uint8(32 + i*20 + j + 1)
			node_set[i][j] = NewKademliaWithId(address, tempID)
		}
	}

	//initialize instance2's k-bucket
	// Initialize node1's k-bucket
	host, port, _ := StringToIpPort("localhost:8101")
	instance3.DoPing(host, port)
	host, port, _ = StringToIpPort("localhost:8121")
	instance3.DoPing(host, port)
	host, port, _ = StringToIpPort("localhost:8141")
	instance3.DoPing(host, port)

	// Initialize three points' k bucket.
	for i := 0; i < 3; i++ {
		for j := 0; j < 20; j++ {
			address := "localhost:" + strconv.Itoa(8100+ i*20+(j+1))
			host, port, _ = StringToIpPort(address)
			node_set[i][0].DoPing(host, port)
		}
	}

	target_instance := node_set[0][0]  //localhost:8102

	new_vdoID := NewRandomID()
	vdo := target_instance.Vanish(new_vdoID,data,numberKeys,threshold,timeoutSeconds)
	fmt.Println("vanish done!")

	res_data := instance3.Unvanish(target_instance.NodeID, new_vdoID)

	if target_instance.VDOmap[new_vdoID].AccessKey !=vdo.AccessKey {
		t.Error("T3: The vanish object have not been stored in the DHT map")
	}

	if !bytes.Equal(data, res_data) {
		t.Error("The unvanised data does not match the vanished data@")
	}

}


func TestUnvanish(t *testing.T) {
	id1 := NewRandomID()
	instance4 := NewKademliaWithId("localhost:9100",id1)
	id2 := NewRandomID()
	instance5 := NewKademliaWithId("localhost:9200",id2)

	data1 :=[]byte("Trying singly unvanish")
	numberKeys := byte(20)
	threshold := byte(10)

	host, port, _ := StringToIpPort("localhost:9100")
	instance5.DoPing(host, port)

	newVDOId := NewRandomID()
	vdoFor4 :=instance4.Vanish(newVDOId,data1,numberKeys,threshold,100)

	fmt.Println("For eli_redLine",vdoFor4.AccessKey)

	dataFor4 :=instance4.Unvanish(instance4.NodeID, newVDOId)

	fmt.Println("data",data1)
	fmt.Println("dataFor4",dataFor4)

	if !bytes.Equal(data1, dataFor4) {
		t.Error("Local unvanish failed!")
	}


	dataFor5:=instance5.Unvanish(instance4.NodeID, newVDOId)

	fmt.Println("dataFor5",dataFor5)

	if !bytes.Equal(data1, dataFor5) {
		t.Error("Two unvanish failed!")
	}

}


//// test timeout, since in every epoch, locations is different but the key should be the same
//// if key is different, error occurs
// func TestTimeOut(t *testing.T) {
// 	var id ID
// 	instance3 := NewKademliaWithId("localhost:6100",id)
// 	data :=[]byte("Distributed System")
// 	numberKeys := byte(20)
// 	threshold := byte(10)
// 	timeoutSeconds := 8 * 3600

// 	//generate 60 nodes
// 	// Generate 60 nodes
// 	node_set := make([][]*Kademlia, 3)
// 	for i := 0; i < 3; i++ {
// 		node_set[i] = make([]*Kademlia, 20)
// 		for j := 0; j < 20; j++ {
// 			address := "localhost:" + strconv.Itoa(6100+ i*20+(j+1))
// 			tempID := instance3.SelfContact.NodeID
// 			tempID[19] = uint8(32 + i*20 + j + 1)
// 			node_set[i][j] = NewKademliaWithId(address, tempID)
// 		}
// 	}

// 	//initialize instance2's k-bucket
// 	// Initialize node1's k-bucket
// 	host, port, _ := StringToIpPort("localhost:6101")
// 	instance3.DoPing(host, port)
// 	host, port, _ = StringToIpPort("localhost:6121")
// 	instance3.DoPing(host, port)
// 	host, port, _ = StringToIpPort("localhost:6141")
// 	instance3.DoPing(host, port)

// 	// Initialize three points' k bucket.
// 	for i := 0; i < 3; i++ {
// 		for j := 0; j < 20; j++ {
// 			address := "localhost:" + strconv.Itoa(6100+ i*20+(j+1))
// 			host, port, _ = StringToIpPort(address)
// 			node_set[i][0].DoPing(host, port)
// 		}
// 	}

// 	target_instance := node_set[0][0]  //localhost:8102

// 	new_vdoID := NewRandomID()
// 	_ = target_instance.Vanish(new_vdoID,data,numberKeys,threshold,timeoutSeconds)
// 	fmt.Println("vanish done!")
// 	// test timeout, since in every epoch, locations is different but the key should be the same
// 	// if key is different, error occurs

// 	for i := 0; i < 3; i ++ {
// 		res_data := instance3.Unvanish(target_instance.NodeID, new_vdoID)

// 		if !bytes.Equal(data, res_data) {
// 			t.Error("The unvanised data does not match the vanished data@")
// 		}
// 		time.Sleep(20000 * time.Millisecond)
// 	}

// }
