package libkademlia


import (
// "net"
"strconv"
"testing"
"net"
"bytes"

)

// Normal Case
func TestIterativeFindNode1(t *testing.T) {
	var id ID
	node1 := NewKademliaWithId("localhost:8000", id)

	// Generate 60 nodes
	node_set := make([][]*Kademlia, 3)
	for i := 0; i < 3; i++ {
		node_set[i] = make([]*Kademlia, 20)
		for j := 0; j < 20; j++ {
			address := "localhost:" + strconv.Itoa(8000+i*20+(j+1))
			tempID := node1.SelfContact.NodeID
			tempID[19] = uint8(32 + i * 20 + j)
			node_set[i][j] = NewKademliaWithId(address, tempID)
		}
	}
	// Initialize node1's k-bucket
	host, port, _ := StringToIpPort("localhost:8001")
	node1.DoPing(host, port)
	host, port, _ = StringToIpPort("localhost:8021")
	node1.DoPing(host, port)
	host, port, _ = StringToIpPort("localhost:8041")
	node1.DoPing(host, port)

	// Initialize three points' k bucket.
	for i := 0; i < 3; i++ {
		for j := 0; j < 20; j++ {
			address := "localhost:" + strconv.Itoa(8000+i*20+(j+1))
			host, port, _ = StringToIpPort(address)
			node_set[i][0].DoPing(host, port)
		}
	}

	keyID := node1.NodeID;
	keyID[19] = uint8(32)
	foundContacts, err := node1.DoIterativeFindNode(keyID);

	if err != nil {
		t.Error("Error: ", err)
	}

	isFound := false
	for i := 0; i < 20; i++ {
		if foundContacts[i].NodeID.Compare(node_set[0][0].NodeID) == 0{
			isFound = true
		}
	}
	if !isFound {
		t.Error("Incorrect result")
	}

	isFound = false
	for i := 0; i < 20; i++ {
		if foundContacts[i].NodeID.Compare(node_set[1][0].NodeID) == 0{
			isFound = true
		}
	}
	if !isFound {
		t.Error("Incorrect result")
	}

	isFound = false
	for i := 0; i < 20; i++ {
		if foundContacts[i].NodeID.Compare(node_set[2][0].NodeID) == 0{
			isFound = true
		}
	}
	if !isFound {
		t.Error("Incorrect result")
	}
}

// One dead initial contact
// Contacts from k-buckects for first acive point should be returned.
func TestIterativeFindNode2(t *testing.T) {
	var id ID
	node1 := NewKademliaWithId("localhost:8200", id)

	// Generate one fake Node
	ipAddrStrings, _ := net.LookupHost("localhost")
	var host net.IP
	for i := 0; i < len(ipAddrStrings); i++ {
		host = net.ParseIP(ipAddrStrings[i])
		if host.To4() != nil {
			break
		}
	}
	tempID := node1.NodeID
	tempID[19] = uint8(32)
	fake_node := Contact{tempID, host, uint16(8201)}
	node1.Table.KBuckets[154].PushBack(fake_node)  //push it to node1's k-buckets

	// Generate 40 nodes
	node_set := make([][]*Kademlia, 3)
	for i := 0; i < 2; i++ {
		node_set[i] = make([]*Kademlia, 20)
		for j := 0; j < 20; j++ {
			address := "localhost:" + strconv.Itoa(8200+(i+1)*20+(j+1))
			tempID := node1.SelfContact.NodeID
			tempID[19] = uint8(32 + (i+ 1) * 20 + j)
			node_set[i][j] = NewKademliaWithId(address, tempID)
		}
	}

	// Initialize node1's k-bucket
	host, port, _ := StringToIpPort("localhost:8221")
	node1.DoPing(host, port)
	host, port, _ = StringToIpPort("localhost:8241")
	node1.DoPing(host, port)

	keyID := node1.NodeID;
	keyID[19] = uint8(32)
	for i := 0; i < 2; i++ {
		for j := 0; j < 20; j++ {
			address := "localhost:" + strconv.Itoa(8200+(i+1)*20+(j+1))
			host, port, _ = StringToIpPort(address)
			node_set[i][0].DoPing(host, port)
		}
	}

	foundContacts, err := node1.DoIterativeFindNode(keyID);

	if err != nil {
		t.Error("Error: ", err)
	}

	isFound := false
	for i := 0; i < 20; i++ {
		if foundContacts[i].NodeID.Compare(node_set[0][0].NodeID) == 0{
			isFound = true
		}
	}
	if !isFound {
		t.Error("Incorrect result")
	}

	isFound = false
	for i := 0; i < 20; i++ {
		if foundContacts[i].NodeID.Compare(node_set[1][0].NodeID) == 0{
			isFound = true
		}
	}
	if !isFound {
		t.Error("Incorrect result")
	}
}

// Three dead initial contacts
// Empty contact list should be returned
func TestIterativeFindNode3(t *testing.T) {
	var id ID
	node1 := NewKademliaWithId("localhost:8300", id)

	// Generate 3 fake nodes
	ipAddrStrings, _ := net.LookupHost("localhost")
	var host net.IP
	for i := 0; i < len(ipAddrStrings); i++ {
		host = net.ParseIP(ipAddrStrings[i])
		if host.To4() != nil {
			break
		}
	}
	for i := 0; i < 3; i++ {
		tempID := node1.NodeID
		tempID[19] = uint8(32 + i)
		temp_node := Contact{tempID, host, uint16(8301 + i)}
		node1.Table.KBuckets[154].PushBack(temp_node)  //push it to node1's k-buckets
	}

	keyID := node1.NodeID;
	keyID[19] = uint8(32)
	foundContacts, err := node1.DoIterativeFindNode(keyID);

	if err == nil {
		t.Error("Error: ", err)
	}

	if len(foundContacts) != 0 {
		t.Error("Incorrect closest list of contacts.")
	}
}

func TestIterativeStroeValue(t *testing.T) {
	var id ID
	node1 := NewKademliaWithId("localhost:7000", id)

	// Generate 60 nodes
	node_set := make([][]*Kademlia, 3)
	for i := 0; i < 3; i++ {
		node_set[i] = make([]*Kademlia, 20)
		for j := 0; j < 20; j++ {
			address := "localhost:" + strconv.Itoa(7000+i*20+(j+1))
			tempID := node1.SelfContact.NodeID
			tempID[19] = uint8(32 + i*20 + j + 1)
			node_set[i][j] = NewKademliaWithId(address, tempID)
		}
	}
	// Initialize node1's k-bucket
	host, port, _ := StringToIpPort("localhost:7001")
	node1.DoPing(host, port)
	host, port, _ = StringToIpPort("localhost:7021")
	node1.DoPing(host, port)
	host, port, _ = StringToIpPort("localhost:7041")
	node1.DoPing(host, port)

	// Initialize three points' k bucket.
	for i := 0; i < 3; i++ {
		for j := 0; j < 20; j++ {
			address := "localhost:" + strconv.Itoa(7000+i*20+(j+1))
			host, port, _ = StringToIpPort(address)
			node_set[i][0].DoPing(host, port)
		}
	}

	key := NewRandomID()
	value := []byte("Hello world")
	// Store the value at the last node
	contacts, err := node1.DoIterativeStore(key, value)
	if err != nil {
		t.Error("Could not store value")
	}

	for i := 0; i < len(contacts); i++ {
		foundValue, _, err := node1.DoFindValue(&contacts[i], key)
		if err != nil {
			t.Error("Could not find value in contacts")
		}
		if !bytes.Equal(foundValue, value) {
			t.Error("Stored value not match")
		}
	}
}

// Normal case
// Value stored at the last node
func TestIterativeFindValue(t *testing.T) {
	var id ID
	node1 := NewKademliaWithId("localhost:7100", id)

	// Generate 60 nodes
	node_set := make([][]*Kademlia, 3)
	for i := 0; i < 3; i++ {
		node_set[i] = make([]*Kademlia, 20)
		for j := 0; j < 20; j++ {
			address := "localhost:" + strconv.Itoa(7100+ i*20+(j+1))
			tempID := node1.SelfContact.NodeID
			tempID[19] = uint8(32 + i*20 + j + 1)
			node_set[i][j] = NewKademliaWithId(address, tempID)
		}
	}
	// Initialize node1's k-bucket
	host, port, _ := StringToIpPort("localhost:7101")
	node1.DoPing(host, port)
	host, port, _ = StringToIpPort("localhost:7121")
	node1.DoPing(host, port)
	host, port, _ = StringToIpPort("localhost:7141")
	node1.DoPing(host, port)

	// Initialize three points' k bucket.
	for i := 0; i < 3; i++ {
		for j := 0; j < 20; j++ {
			address := "localhost:" + strconv.Itoa(7100+ i*20+(j+1))
			host, port, _ = StringToIpPort(address)
			node_set[i][0].DoPing(host, port)
		}
	}

	key := NewRandomID()
	value := []byte("Hello world")
	// Store the value at the last node
	err := node_set[2][19].DoStore(&node_set[2][19].SelfContact, key, value)
	if err != nil {
		t.Error("Could not store value")
	}

	// Given the right keyID, it should return the value
	foundValue, err := node1.DoIterativeFindValue(key)
	println("see here:",string(foundValue))
	if err != nil {

		t.Error("Cound not find value")
	}


	for i := 0; i < len(foundValue); i++ {
		println(foundValue[i])
	}
	if !bytes.Equal(foundValue, value) {
		t.Error("Stored value did not match found value")
	}

	if !bytes.Equal(node_set[0][0].DataMap[key], value) {
		t.Error("Closest node with didn't store the value, still doesn't store it after iterativeFindValue")
	}

	//Given the wrong keyID, it should return k nodes.
	wrongKey := NewRandomID()
	if err != nil {
		t.Error("Cound not find value")
	}
	foundValue, err = node1.DoIterativeFindValue(wrongKey)
	if foundValue == nil {
		t.Error("Searching for a wrong ID did not return contacts")
	}
}
