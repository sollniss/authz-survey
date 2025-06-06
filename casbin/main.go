package main

import (
	"fmt"
	"log"

	"github.com/casbin/casbin/v2"
)

func main() {
	// Initialize Casbin enforcer
	enforcer, err := casbin.NewEnforcer("model.conf", "policy.csv")
	if err != nil {
		log.Fatal("load data: " + err.Error())
	}

	// Helper function to check access and print result
	checkAccess := func(user, action, resource string) {
		ok, err := enforcer.Enforce(user, action, resource)
		if err != nil {
			log.Fatal("enforce: " + err.Error())
		}

		result := "DENIED"
		if ok {
			result = "ALLOWED"
		}

		fmt.Printf("%-8s trying to %-6s %-10s: %s\n", user, action, resource, result)
	}

	fmt.Println("\n=== STANDARD COMPANY ACCESS PATTERNS ===")
	// Users accessing their own company's products
	checkAccess("alice", "read", "product1")
	checkAccess("alice", "write", "product1")
	checkAccess("alice", "read", "product2")
	checkAccess("alice", "write", "product2")
	checkAccess("bob", "read", "product2")
	checkAccess("bob", "write", "product2")

	fmt.Println("\n=== CROSS-COMPANY ACCESS SCENARIOS ===")
	// Charlie from company1 accessing company2's product4 (special case)
	checkAccess("charlie", "read", "product3")
	checkAccess("charlie", "write", "product3")
	checkAccess("charlie", "read", "product4")
	checkAccess("charlie", "read", "product5")

	fmt.Println("\n=== DAVE'S SPECIAL PERMISSIONS ===")
	// Dave from company2 with read access to company1 but denied for product2
	checkAccess("dave", "read", "product1")
	checkAccess("dave", "write", "product2")
	checkAccess("dave", "read", "product2")
	checkAccess("dave", "read", "product3")

	fmt.Println("\n=== PERMISSION HIERARCHY TESTS ===")
	// Testing if write permission includes read permission
	checkAccess("eve", "write", "product5")
	checkAccess("eve", "read", "product5")

	fmt.Println("\n=== ADDING NEW USERS AND PERMISSIONS ===")

	// Add a new user, company, and product
	_, err = enforcer.AddGroupingPolicy("frank", "company3")
	if err != nil {
		log.Fatal("add frank grouping policy: " + err.Error())
	}
	_, err = enforcer.AddGroupingPolicy("product7", "company3")
	if err != nil {
		log.Fatal("add product7 grouping policy: " + err.Error())
	}
	_, err = enforcer.AddPolicy("company3", "read", "product7", "allow")
	if err != nil {
		log.Fatal("add company3 read policy: " + err.Error())
	}
	_, err = enforcer.AddPolicy("frank", "write", "product7", "allow")
	if err != nil {
		log.Fatal("add frank write policy: " + err.Error())
	}

	// Test new permissions
	checkAccess("frank", "read", "product7")
	checkAccess("frank", "write", "product7")
	checkAccess("alice", "read", "product7")

	fmt.Println("\n=== DYNAMIC POLICY CHANGES ===")
	// Give Alice access to Frank's product
	_, err = enforcer.AddGroupingPolicy("alice", "product7", "read")
	if err != nil {
		log.Fatal("add alice read policy for product7: " + err.Error())
	}
	checkAccess("alice", "read", "product7")
}
