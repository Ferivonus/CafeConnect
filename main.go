package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

type CoffeeShop struct {
	Name  string
	Price float64
}

type CoffeeCount struct {
	Name         string
	Count        int
	InitialName  string
	InitialCount int
}

func main() {
	db := connectToDatabase()
	defer db.Close()

	http.HandleFunc("/admin", handleAdminPage(db))
	http.HandleFunc("/index", handleHomePage(db))

	fmt.Println("Server starting on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func connectToDatabase() *sql.DB {
	db, err := sql.Open("mysql", "root:password@/")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS coffeeshops_db")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec("USE coffeeshops_db")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS coffeeshops (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(50) NOT NULL,
		price FLOAT(10,2) NOT NULL
	)`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS coffee_counts (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(50) NOT NULL,
		count INT NOT NULL,
		initial_name VARCHAR(50) NOT NULL,
		initial_count INT NOT NULL
	)`)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func handleAdminPage(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			coffeeShops := getAllCoffeeShops(db)
			coffeeCounts := getCoffeeCounts(db)
			data := struct {
				CoffeeShops  []CoffeeShop
				CoffeeCounts []CoffeeCount
			}{coffeeShops, coffeeCounts}
			adminPageTemplate.Execute(w, data)
		case "POST":
			if r.URL.Path == "/admin/add" {
				name := r.FormValue("name")
				priceStr := r.FormValue("price")
				price, err := strconv.ParseFloat(priceStr, 64)
				if err != nil {
					http.Error(w, "Invalid price", http.StatusBadRequest)
					return
				}
				countStr := r.FormValue("count")
				count, err := strconv.Atoi(countStr)
				if err != nil {
					http.Error(w, "Invalid count", http.StatusBadRequest)
					return
				}
				addNewCoffeeShop(db, name, price)
				incrementCoffeeCount(db, name, count)
			} else if r.URL.Path == "/admin/update" {
				countStr := r.FormValue("count")
				count, err := strconv.Atoi(countStr)
				if err != nil {
					http.Error(w, "Invalid count", http.StatusBadRequest)
					return
				}
				updateCoffeeCount(db, r.FormValue("name"), count, w)
			} else if r.URL.Path == "/admin/delete" {
				deleteCoffeeShop(db, r.FormValue("name"))
				deleteCoffeeCount(db, r.FormValue("name"))
			}
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func getAllCoffeeShops(db *sql.DB) []CoffeeShop {
	rows, err := db.Query("SELECT name, price FROM coffeeshops")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var coffeeShops []CoffeeShop
	for rows.Next() {
		var cs CoffeeShop
		if err := rows.Scan(&cs.Name, &cs.Price); err != nil {
			log.Fatal(err)
		}
		coffeeShops = append(coffeeShops, cs)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return coffeeShops
}

func getCoffeeCounts(db *sql.DB) []CoffeeCount {
	rows, err := db.Query("SELECT name, count, initial_name, initial_count FROM coffee_counts")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var coffeeCounts []CoffeeCount
	for rows.Next() {
		var cc CoffeeCount
		if err := rows.Scan(&cc.Name, &cc.Count, &cc.InitialName, &cc.InitialCount); err != nil {
			log.Fatal(err)
		}
		coffeeCounts = append(coffeeCounts, cc)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return coffeeCounts
}

func addNewCoffeeShop(db *sql.DB, name string, price float64) {
	_, err := db.Exec("INSERT INTO coffeeshops (name, price) VALUES (?, ?)", name, price)
	if err != nil {
		log.Fatal(err)
	}

}

func incrementCoffeeCount(db *sql.DB, name string, count int) {
	stmt, err := db.Prepare("INSERT INTO coffee_counts (name, count, initial_name, initial_count) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE count = count + ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	initial := getCoffeeCount(db, name)
	_, err = stmt.Exec(name, count, initial.Name, initial.Count, count)
	if err != nil {
		log.Fatal(err)
	}
}

func updateCoffeeCount(db *sql.DB, name string, count int, w http.ResponseWriter) {
	result, err := db.Exec("UPDATE coffee_counts SET count = ? WHERE name = ?", count, name)
	if err != nil {
		http.Error(w, "Failed to update coffee count", http.StatusInternalServerError)
		log.Fatal(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatal(err)
	}

	if rowsAffected == 0 {
		http.Error(w, "No coffee count updated", http.StatusNotFound)
		return
	}
}

func deleteCoffeeShop(db *sql.DB, name string) {
	_, err := db.Exec("DELETE FROM coffeeshops WHERE name = ?", name)
	if err != nil {
		log.Fatal(err)
	}
}

func deleteCoffeeCount(db *sql.DB, name string) {
	_, err := db.Exec("DELETE FROM coffee_counts WHERE name = ?", name)
	if err != nil {
		log.Fatal(err)
	}
}

func getCoffeeCount(db *sql.DB, name string) CoffeeCount {
	row := db.QueryRow("SELECT name, count, initial_name, initial_count FROM coffee_counts WHERE name = ?", name)

	var cc CoffeeCount
	err := row.Scan(&cc.Name, &cc.Count, &cc.InitialName, &cc.InitialCount)
	if err == sql.ErrNoRows {
		cc.Name = name
		cc.Count = 0
	} else if err != nil {
		log.Fatal(err)
	}

	return cc
}

func handleHomePage(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			coffeeCounts := getCoffeeCounts(db)
			initialCoffeeCounts := getInitialCoffeeCounts(db)
			data := struct {
				CoffeeCounts        []CoffeeCount
				InitialCoffeeCounts []CoffeeCount
			}{coffeeCounts, initialCoffeeCounts}
			homePageTemplate.Execute(w, data)
		case "POST":
			updateCoffeeCount(db, r.FormValue("name"), 0, w)
			http.Redirect(w, r, "/index", http.StatusSeeOther)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func getInitialCoffeeCounts(db *sql.DB) []CoffeeCount {
	rows, err := db.Query("SELECT name, count, initial_name, initial_count FROM coffee_counts WHERE count = initial_count")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var coffeeCounts []CoffeeCount
	for rows.Next() {
		var cc CoffeeCount
		if err := rows.Scan(&cc.Name, &cc.Count, &cc.InitialName, &cc.InitialCount); err != nil {
			log.Fatal(err)
		}
		coffeeCounts = append(coffeeCounts, cc)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	return coffeeCounts
}

func getInitialCoffeeCount(name string, initialCoffeeCounts []CoffeeCount) int {
	for _, cc := range initialCoffeeCounts {
		if cc.InitialName == name {
			return cc.InitialCount
		}
	}
	return 0
}

var adminPageTemplate = template.Must(template.New("admin").Funcs(template.FuncMap{
	"getInitialCoffeeCount": getInitialCoffeeCount,
}).ParseFiles("html files/admin.html"))

var homePageTemplate = template.Must(template.New("home").Funcs(template.FuncMap{
	"getInitialCoffeeCount": getInitialCoffeeCount,
}).ParseFiles("html files/index.html"))
