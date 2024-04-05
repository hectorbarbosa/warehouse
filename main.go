package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"warehouse/database"
)

type InvoiceRow struct {
	OrderID     int64
	ProductID   int32
	Quantity    uint32
	ProductName string
	ShelfName   string
	AddShelves  string
}

func main() {
	// логгер
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}
	log.SetOutput(file)

	// аргументы командной строки
	input := strings.Join(os.Args[1:], " ")
	ordersString := strings.ReplaceAll(input, " ", "")

	// todo!
	// сделать проверку пользовательского ввода

	// Открываем базу данных
	db, err := database.GetDB()
	if err != nil {
		log.Fatal(err)
	}

	// Строки накладной, 3 поля
	invoiceShortRows, err := database.GetInvoiceRows(db, ordersString)
	if err != nil {
		log.Fatal(err)
	}

	// product ID из слайса в строку для запроса в базу
	productsID := getProductsID(invoiceShortRows)

	// названия товаров в строку
	productNames, err := database.GetProductNames(db, productsID)
	if err != nil {
		log.Fatal(err)
	}

	mainShelvesID, err := database.GetMainShelvesID(db, productsID)
	if err != nil {
		log.Fatal(err)
	}

	// ID главных стеллажей из слайса в строку для запроса в базу
	stringShelves := getShelvesID(mainShelvesID)
	// Буквенные названия стеллажей
	sortedShelves, shelfNames, err := database.GetMainShelfNames(db, stringShelves)
	if err != nil {
		log.Fatal(err)
	}

	// доп. стеллажи в строку
	addNames, err := database.GetAddShelves(db, productsID)
	if err != nil {
		log.Fatal(err)
	}

	// -------------------------------------------------------
	// Собираем все в одну накладную
	var invoice []*InvoiceRow
	for _, shelfID := range sortedShelves {

		for _, v := range invoiceShortRows {
			if s := mainShelvesID[v.ProductID]; s == shelfID {
				row := InvoiceRow{}
				row.OrderID = v.OrderID
				row.ProductID = v.ProductID
				row.Quantity = v.Quantity
				row.ProductName = productNames[v.ProductID]
				row.ShelfName = shelfNames[shelfID]
				row.AddShelves = addNames[v.ProductID]
				invoice = append(invoice, &row)
			}
		}
	}

	// печатаем накладную
	printInvoice(invoice, ordersString)
}

func printInvoice(invoice []*InvoiceRow, orders string) {
	fmt.Println("=+=+=+=")
	fmt.Printf("Страница сборки заказов %s\n\n\n", orders)
	shelf := ""
	for _, row := range invoice {
		if row.ShelfName != shelf {
			shelf = row.ShelfName
			fmt.Printf("===Стеллаж %s \n", shelf)
		}
		fmt.Printf("%s (id=%d)\n", row.ProductName, row.ProductID)
		fmt.Printf("заказ %d, %d шт\n", row.OrderID, row.Quantity)
		if row.AddShelves != "" {
			fmt.Printf("доп стеллаж: %s\n", row.AddShelves)
		}
		fmt.Println("")
	}
}

func getShelvesID(shelves map[int32]int32) string {
	var s string
	for _, v := range shelves {
		if len(s) == 0 {
			s += strconv.Itoa(int(v))
		} else {
			substr := strconv.Itoa(int(v))
			if strings.Contains(s, substr) == false {
				s += "," + substr
			}
		}

	}
	return s
}

func getAddShelvesID(shelves map[int32][]int32) string {
	var s string
	for _, productID := range shelves {
		for _, shelfID := range productID {
			if len(s) == 0 {
				s += strconv.Itoa(int(shelfID))
			} else {
				substr := strconv.Itoa(int(shelfID))
				if strings.Contains(s, substr) == false {
					s += "," + substr
				}
			}
		}
	}
	return s
}

func getProductsID(invoiceShortRows []*database.InvoiceShort) string {
	var p string
	for _, v := range invoiceShortRows {
		if len(p) == 0 {
			p += strconv.Itoa(int(v.ProductID))
		} else {
			substr := strconv.Itoa(int(v.ProductID))
			if strings.Contains(p, substr) == false {
				p += "," + substr
			}
		}
	}

	return p
}
