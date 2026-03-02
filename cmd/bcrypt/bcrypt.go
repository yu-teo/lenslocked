package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	switch os.Args[1] {
	case "hash":
		hash(os.Args[2])
	case "compare":
		compare(os.Args[2], os.Args[3])
	default:
		fmt.Println("Invalid command: %v\n", os.Args[1])
	}
}

func hash(password string) {
	fmt.Printf("You are trying to hash: %q\n", password)
	pass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Something went wrong when hashing the password")
	}
	fmt.Println("Successfully hashed into: ", string(pass))
}

func compare(password string, hashValue string) {
	fmt.Printf("You are trying to compare a password: %q with a hash: %q\n", password, hashValue)
	err := bcrypt.CompareHashAndPassword([]byte(hashValue), []byte(password))
	if err != nil {
		fmt.Println("Something went wrong when comparing the hash to the password")
		return
	}
	fmt.Println("Password is correct!")
}
