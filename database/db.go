package database

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

type InvoiceShort struct {
	ProductID int32
	OrderID   int64
	Quantity  uint32
}

const (
	host     = "localhost"
	port     = 5432
	user     = "u"
	password = "pass"
	dbname   = "warehouse"
)

func GetDB() (*sql.DB, error) {
	psqlUser := fmt.Sprintf("host=%s port=%d user=%s password=%s "+
		"dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlUser)
	if err != nil {
		return nil, err
	}
	return db, err
}

func GetInvoiceRows(db *sql.DB, orders string) ([]*InvoiceShort, error) {
	query := fmt.Sprintf(`
        SELECT
            w.order_content.order_id,
            w.order_content.product_id,
            w.order_content.quantity
        FROM
            w.order_content
        WHERE
            w.order_content.order_id in (%s);`,
		orders)
	// log.Println(query)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoice []*InvoiceShort
	// Проходим по строкам заказов
	for rows.Next() {
		var invoiceRow InvoiceShort
		err := rows.Scan(
			&invoiceRow.OrderID,
			&invoiceRow.ProductID,
			&invoiceRow.Quantity,
		)
		if err != nil {
			return nil, err
		}
		invoice = append(invoice, &invoiceRow)
	}

	return invoice, nil
}

// функция возвращает названия товаров
func GetProductNames(db *sql.DB, productIDs string) (map[int32]string, error) {
	var productNames = make(map[int32]string)
	query := fmt.Sprintf(`
		SELECT
			w.products.product_id,
            w.products.product_name

		FROM
			w.products
		WHERE
			w.products.product_id in (%s);`,
		productIDs)
	// log.Println(query)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var id int32
	var name string
	for rows.Next() {
		err = rows.Scan(&id, &name)
		if err != nil {
			return nil, err
		}

		productNames[id] = name
	}

	return productNames, nil
}

func GetMainShelvesID(db *sql.DB, productsID string) (
	map[int32]int32, // главный стеллаж только один, поэтому достаточно словаря
	error,
) {
	var mainShelf = make(map[int32]int32)

	query := fmt.Sprintf(`
		SELECT
			w.shelf_content.shelf_id,
            w.shelf_content.product_id
		FROM
			w.shelf_content
		WHERE
			w.shelf_content.product_id in (%s) AND w.shelf_content.main_shelf = true;`,
		productsID)
	// log.Println(query)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shelfID, productID int32
	for rows.Next() {
		err = rows.Scan(&shelfID, &productID)
		if err != nil {
			return nil, err
		}
		// главный стеллаж только один
		mainShelf[productID] = shelfID
	}

	return mainShelf, nil
}

// Названия дополнительных шкафов из ID товара
func GetAddShelves(db *sql.DB, productsID string) (
	map[int32]string, // дополнительных стеллажей может быть несколько
	error,
) {
	var mapShelvesID = make(map[int32][]int32)

	query := fmt.Sprintf(`
		SELECT
			w.shelf_content.shelf_id,
            w.shelf_content.product_id
		FROM
			w.shelf_content
		WHERE
			w.shelf_content.product_id in (%s) AND w.shelf_content.main_shelf=false;`,
		productsID)

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shelfID, productID int32
	for rows.Next() {
		err = rows.Scan(&shelfID, &productID)
		if err != nil {
			return nil, err
		}
		// дополнительных стеллажей может быть несколько
		// добавляем их в слайс
		if len(mapShelvesID[productID]) == 0 {

			newSlice := make([]int32, 2)
			mapShelvesID[productID] = newSlice
		}
		mapShelvesID[productID] = append(mapShelvesID[productID], shelfID)
	}

	var shelves []int32
	// ID доп шкафов для запроса
	for _, v := range mapShelvesID {
		shelves = append(shelves, v...)
	}

	shelvesID := strings.Trim(strings.Join(strings.Split(fmt.Sprint(shelves), " "), ","), "[]")

	query = fmt.Sprintf(`
    SELECT
        w.shelves.shelf_id,
        w.shelves.shelf_name
    FROM
        w.shelves
    WHERE
        w.shelves.shelf_id in (%s)
    ORDER BY
        w.shelves.shelf_name;`,
		shelvesID)

	rows, err = db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	namesMap := make(map[int32]string)
	for rows.Next() {
		var shelfID int32
		var shelfName string
		err := rows.Scan(&shelfID, &shelfName)
		if err != nil {
			return nil, err
		}
		namesMap[shelfID] = shelfName
	}
	// log.Printf("Add: %v", mapNames)

	resultMap := make(map[int32]string)

	for prod, shelves := range mapShelvesID {
		var s string
		for _, shelfID := range shelves {
			if len(s) == 0 {
				s += namesMap[shelfID]
			} else {
				s += "," + namesMap[shelfID]
			}

		}
		resultMap[prod] = s
	}

	return resultMap, nil
}

// функция возвращает отсортированный слайс и мапу с названиями стеллажей
func GetMainShelfNames(db *sql.DB, shelfIDs string) ([]int32, map[int32]string, error) {
	query := fmt.Sprintf(`
    SELECT
        w.shelves.shelf_id,
        w.shelves.shelf_name
    FROM
        w.shelves
    WHERE
        w.shelves.shelf_id in (%s)
    ORDER BY
        w.shelves.shelf_name;`,
		shelfIDs)
	rows, err := db.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	sliceNames := make([]int32, 10) // отсортированный массив
	mapNames := make(map[int32]string)
	for rows.Next() {
		var id int32
		var name string
		err := rows.Scan(&id, &name)
		if err != nil {
			return nil, nil, err
		}
		mapNames[id] = name
		sliceNames = append(sliceNames, id)
	}
	// log.Printf("main: %v", mapNames)
	return sliceNames, mapNames, nil
}
