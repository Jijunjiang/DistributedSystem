package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"//"net/rpc"
	"os"
	"strconv"
	"strings"
	"time"
)

import (
	"libkademlia"
)

func main() {
	// TODO: PUT YOUR GROUP'S NET IDS HERE!
	// Example:
	// netIds := "abc123 def456 ghi789"
	netIds := "slq0189 jjp1083 ycn2555s"
	if len(netIds) == 0 {
		log.Fatal("Variable containing group's net IDs is not set!\n")
	}

	// By default, Go seeds its RNG with 1. This would cause every program to
	// generate the same sequence of IDs. Use the current nano time to
	// random numbers
	rand.Seed(time.Now().UnixNano())

	// Get the bind and connect connection strings from command-line arguments.
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		log.Fatal("Must be invoked with exactly two arguments!\n")
	}
	listenStr := args[0]
	firstPeerStr := args[1]

	// Create the Kademlia instance
	log.Println("Kademlia starting up!")
	log.Println("Group: " + netIds + "\n")

	kadem := libkademlia.NewKademlia(listenStr)

	// Confirm our server is up with a PING request and then exit.
	// Your code should loop forever, reading instructions from stdin and
	// printing their results to stdout. See README.txt for more details.
	hostname, ports, err := net.SplitHostPort(firstPeerStr)
	
	
	port, err := strconv.Atoi(ports)
	
	if err != nil {
		log.Fatal("DialHTTP: ", err)
	}
	ipAddrStrings, err := net.LookupHost(hostname)
	var host net.IP
			for i := 0; i < len(ipAddrStrings); i++ {
				host = net.ParseIP(ipAddrStrings[i])
				if host.To4() != nil {
					break
				}
			}

	log.Printf("Pinging initial peer\n")

	// This is a sample of what an RPC looks like
	// TODO: Replace this with a call to your completed DoPing!

	kadem.DoPing(host, uint16(port))
	// ping := new(libkademlia.PingMessage)
	// ping.MsgID = libkademlia.NewRandomID()
	// var pong libkademlia.PongMessage
	// err = client.Call("KademliaRPC.Ping", ping, &pong)
	// if err != nil {
	// 	log.Fatal("Call: ", err)
	// }
	// log.Printf("ping msgID: %s\n", ping.MsgID.AsString())
	// log.Printf("pong msgID: %s\n\n", pong.MsgID.AsString())

	in := bufio.NewReader(os.Stdin)
	quit := false
	for !quit {
		fmt.Printf("kademlia> ")
		line, err := in.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		resp := executeLine(kadem, line)
		if resp == "quit" {
			quit = true
		} else if resp != "" {
			fmt.Printf("%v\n", resp)
		}
	}
}

func executeLine(k *libkademlia.Kademlia, line string) (response string) {
	toks := strings.Fields(line)
	switch {
	case toks[0] == "quit":
		response = "quit"

	case toks[0] == "exit":
		response = "quit"

	case toks[0] == "whoami":
		if len(toks) > 1 {
			response = "usage: whoami"
			return
		}
		response = k.NodeID.AsString()

	case toks[0] == "print_contact":
		if len(toks) < 2 || len(toks) > 2 {
			response = "usage: print_contact [nodeID]"
			return
		}
		id, err := libkademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Not a valid node ID (" + toks[1] + ")"
			return
		}
		c, err := k.FindContact(id)
		if err != nil {
			response = "ERR: Unknown contact node ID"
			return
		}
		response = "OK: NodeID=" + toks[1] + "\n"
		response += "      Host=" + c.Host.String() + "\n"
		response += "      Port=" + strconv.Itoa(int(c.Port))

	case toks[0] == "ping":
		// Do a ping
		//
		// Check if toks[1] is a valid NodeID, if not, try pinging host:port
		// print an error if neither is valid
		//
		// Following lines need to be expanded
		var contact *libkademlia.Contact = nil
		var err error

		if len(toks) < 2 || len(toks) > 2 {
			response = "usage: ping [nodeID | host:port]"
			return
		}
		id, err := libkademlia.IDFromString(toks[1])
		if err != nil {
			hostname, portstr, err := net.SplitHostPort(toks[1])
			if err != nil {
				response = "ERR: Not a valid Node ID or host:port address"
				return
			}
			port, err := strconv.Atoi(portstr)
			if err != nil {
				response = "ERR: Not a valid Node ID or host:port address"
				return
			}
			ipAddrStrings, err := net.LookupHost(hostname)
			if err != nil {
				response = "ERR: Could not find the provided hostname"
				return
			}
			var host net.IP
			for i := 0; i < len(ipAddrStrings); i++ {
				host = net.ParseIP(ipAddrStrings[i])
				if host.To4() != nil {
					break
				}
			}
			fmt.Println(host, port)
			contact, err = k.DoPing(host, uint16(port))
			if err != nil {
				response = fmt.Sprintf("ERR: %s", err)
				return
			} else {
				response = "OK: " + contact.NodeID.AsString()
				return
			}
		} else {
			c, err := k.FindContact(id)
			if err != nil {
				response = "ERR: Not a valid Node ID or host:port address"
				return
			}
			contact, err = k.DoPing(c.Host, c.Port)
			if err != nil {
				response = fmt.Sprintf("ERRoo: %s", err)
			} else {
				response = "OK: " + contact.NodeID.AsString()
			}
		}

	case toks[0] == "local_find_value":
		// print a local variable
		if len(toks) < 2 || len(toks) > 2 {
			response = "usage: local_find_value [key]"
			return
		}
		key, err := libkademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid key (" + toks[1] + ")"
			return
		}
		result, err := k.LocalFindValue(key)
		if err != nil {
			response = fmt.Sprintf("ERR: %s", err)
		} else {
			response = "OK: " + string(result)
		}

	case toks[0] == "store":
		// Store key, value pair at NodeID
		if len(toks) < 4 || len(toks) > 4 {
			response = "usage: store [nodeID] [key] [value]"
			return
		}
		nodeId, err := libkademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid node ID (" + toks[1] + ")"
			return
		}
		contact, err := k.FindContact(nodeId)
		if err != nil {
			response = "ERR: Unable to find contact with node ID (" + toks[1] + ")"
			return
		}
		key, err := libkademlia.IDFromString(toks[2])
		if err != nil {
			response = "ERR: Provided an invalid key (" + toks[2] + ")"
			return
		}
		value := []byte(toks[3])

		err = k.DoStore(contact, key, value)
		if err != nil {
			response = fmt.Sprintf("ERR: %s", err)
		} else {
			response = fmt.Sprintf("OK: %s stored by contact %s at key %s",
				string(value), nodeId.AsString(), key.AsString())
		}

	case toks[0] == "find_node":
		// perform a find_node RPC
		if len(toks) < 3 || len(toks) > 3 {
			response = "usage: find_node [nodeID] [key]"
			return
		}

		nodeId, err := libkademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid node ID (" + toks[1] + ")"
			return
		}
		contact, err := k.FindContact(nodeId)
		if err != nil {
			response = "ERR: Unable to find contact with node ID (" + toks[1] + ")"
			return
		}
		key, err := libkademlia.IDFromString(toks[2])
		if err != nil {
			response = "ERR: Provided an invalid key (" + toks[2] + ")"
			return
		}
		contacts, err := k.DoFindNode(contact, key)
		if err != nil {
			response = fmt.Sprintf("ERR: %s", err)
		} else {
			response = fmt.Sprintf("Ok: Got %d contacts", len(contacts))
		}

	case toks[0] == "find_value":
		// perform a find_value RPC
		if len(toks) < 3 || len(toks) > 3 {
			response = "usage: find_value [nodeID] [key]"
			return
		}

		nodeId, err := libkademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid node ID (" + toks[1] + ")"
			return
		}
		contact, err := k.FindContact(nodeId)
		if err != nil {
			response = "ERR: Unable to find contact with node ID (" + toks[1] + ")"
			return
		}
		key, err := libkademlia.IDFromString(toks[2])
		if err != nil {
			response = "ERR: Provided an invalid key (" + toks[2] + ")"
			return
		}
		value, contacts, err := k.DoFindValue(contact, key)
		if err != nil {
			response = fmt.Sprintf("ERR: %s", err)
		} else if value != nil {
			response = fmt.Sprintf("OK: Found %s", value)
		} else {
			response = fmt.Sprintf("OK: Got %d contacts", len(contacts))
		}

	case toks[0] == "iterativeFindNode":
		// perform an iterative find node
		if len(toks) != 2 {
			response = "usage: iterativeFindNode [nodeID]"
			return
		}
		id, err := libkademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid node ID(" + toks[1] + ")"
			return
		}
		contacts, err := k.DoIterativeFindNode(id)
		if err != nil {
			response = fmt.Sprintf("ERR: %s", err)
		} else {
			response = fmt.Sprintf("OK: Got %d contacts", len(contacts))
		}

	case toks[0] == "iterativeStore":
		// perform an iterative store
		if len(toks) != 3 {
			response = "usage: iterativeStore [key] [value]"
			return
		}
		key, err := libkademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid key (" + toks[1] + ")"
			return
		}
		contacts, err := k.DoIterativeStore(key, []byte(toks[2]))
		if err != nil {
			response = fmt.Sprintf("ERR: %s", err)
		} else {
			response = fmt.Sprintf("OK: Stored value on %d contacts", len(contacts))
		}

	case toks[0] == "iterativeFindValue":
		// performa an iterative find value
		if len(toks) != 2 {
			response = "usage: iterativeFindValue [key]"
			return
		}
		key, err := libkademlia.IDFromString(toks[1])
		if err != nil {
			response = "ERR: Provided an invalid key (" + toks[1] + ")"
			return
		}
		value, err := k.DoIterativeFindValue(key)
		if err != nil {
			response = fmt.Sprintf("ERR: %s", err)
		} else {
			response = fmt.Sprintf("OK: Found value %s", value)
		}

	default:
		response = "ERR: Unknown command"
	}
	return
}
