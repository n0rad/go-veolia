package veolia

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

//func TestConsumption(t *testing.T) {
//	veolia := NewVeolia()
//	veolia.Username = "XXXXX"
//	veolia.Password = "XXXXX"
//	conso, err := veolia.getConsumption()
//	if err != nil {
//		t.Fatal(err)
//	}
//	for _, e := range conso {
//		fmt.Println(e.Day, " > ", e.Index, " > ", e.Consumption, " > ", e.Type)
//	}
//}

func TestFailedLogin(t *testing.T) {
	veolia := NewVeolia()
	veolia.Username = "XXXXX"
	veolia.Password = "XXXXX"
	_, err := veolia.getConsumption()
	if !strings.Contains(err.Error(), "Login failed") {
		t.Error(err)
	}
}

func TestReadFile(t *testing.T) {
	f, _ := os.Open("test/sample.xls")
	defer f.Close()
	c, _ := ioutil.ReadAll(f)
	conso, _ := readConsumptionXls(c)
	for _, e := range conso {
		fmt.Println(e.Day, " > ", e.Index, " > ", e.Consumption, " > ", e.Type)
	}

}
