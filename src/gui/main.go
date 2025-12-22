package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func main() {
	myApp := app.New()
	mainWindow := myApp.NewWindow("OpenERP - NAV-Style ERP System")
	mainWindow.Resize(fyne.NewSize(1000, 600))

	gui := &GUI{
		app:    myApp,
		window: mainWindow,
	}

	gui.showDatabaseSelection()
	mainWindow.ShowAndRun()
}

type GUI struct {
	app     fyne.App
	window  fyne.Window
	db      *Database
	company string
}

// discoverDatabases scans the current directory for .db files
func discoverDatabases() ([]string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	files, err := filepath.Glob(filepath.Join(currentDir, "*.db"))
	if err != nil {
		return nil, err
	}

	// Convert to just filenames
	var databases []string
	for _, file := range files {
		databases = append(databases, filepath.Base(file))
	}

	return databases, nil
}

// Database Selection Screen
func (g *GUI) showDatabaseSelection() {
	title := widget.NewLabel("Welcome to OpenERP")
	title.TextStyle.Bold = true

	// Discover existing databases
	databases, err := discoverDatabases()
	if err != nil {
		dialog.ShowError(fmt.Errorf("Failed to scan for databases: %w", err), g.window)
		databases = []string{}
	}

	var content *fyne.Container

	if len(databases) > 0 {
		// Show existing databases
		subtitle := widget.NewLabel(fmt.Sprintf("Found %d database(s) in current directory:", len(databases)))

		dbList := widget.NewList(
			func() int { return len(databases) },
			func() fyne.CanvasObject {
				return widget.NewLabel("database.db")
			},
			func(id widget.ListItemID, obj fyne.CanvasObject) {
				obj.(*widget.Label).SetText(databases[id])
			},
		)

		dbList.OnSelected = func(id widget.ListItemID) {
			dbPath := databases[id]
			db := &Database{}
			if err := db.OpenDatabase(dbPath); err != nil {
				dialog.ShowError(fmt.Errorf("Failed to open database: %w", err), g.window)
				return
			}
			g.db = db
			g.showCompanyList()
		}

		// Create new database option
		newDBEntry := widget.NewEntry()
		newDBEntry.SetPlaceHolder("Enter new database name (e.g., myerp.db)")

		createNewBtn := widget.NewButton("Create New Database", func() {
			dbPath := strings.TrimSpace(newDBEntry.Text)
			if dbPath == "" {
				dialog.ShowError(fmt.Errorf("Database name cannot be empty"), g.window)
				return
			}

			// Add .db extension if not present
			if !strings.HasSuffix(dbPath, ".db") {
				dbPath += ".db"
			}

			// Check if file already exists
			if _, err := os.Stat(dbPath); err == nil {
				dialog.ShowError(fmt.Errorf("Database '%s' already exists. Please select it from the list above.", dbPath), g.window)
				return
			}

			db := &Database{}
			if err := db.OpenDatabase(dbPath); err != nil {
				dialog.ShowError(fmt.Errorf("Failed to create database: %w", err), g.window)
				return
			}

			g.db = db
			dialog.ShowInformation("Success", fmt.Sprintf("Database '%s' created successfully", dbPath), g.window)
			g.showCompanyList()
		})

		content = container.NewBorder(
			container.NewVBox(
				title,
				widget.NewLabel(""),
				subtitle,
			),
			container.NewVBox(
				widget.NewSeparator(),
				widget.NewLabel("Or create a new database:"),
				newDBEntry,
				createNewBtn,
			),
			nil,
			nil,
			dbList,
		)
	} else {
		// No databases found - show create new database form
		subtitle := widget.NewLabel("No databases found in current directory")
		subtitle.TextStyle.Italic = true

		dbPathEntry := widget.NewEntry()
		dbPathEntry.SetPlaceHolder("Enter database name (e.g., erp.db)")
		dbPathEntry.SetText("erp.db")

		createBtn := widget.NewButton("Create Database", func() {
			dbPath := strings.TrimSpace(dbPathEntry.Text)
			if dbPath == "" {
				dialog.ShowError(fmt.Errorf("Database name cannot be empty"), g.window)
				return
			}

			// Add .db extension if not present
			if !strings.HasSuffix(dbPath, ".db") {
				dbPath += ".db"
			}

			db := &Database{}
			if err := db.OpenDatabase(dbPath); err != nil {
				dialog.ShowError(fmt.Errorf("Failed to create database: %w", err), g.window)
				return
			}

			g.db = db
			dialog.ShowInformation("Success", fmt.Sprintf("Database '%s' created successfully", dbPath), g.window)
			g.showCompanyList()
		})

		content = container.NewVBox(
			widget.NewLabel(""),
			title,
			widget.NewLabel(""),
			subtitle,
			widget.NewLabel(""),
			widget.NewLabel("Database Name:"),
			dbPathEntry,
			createBtn,
		)
	}

	g.window.SetContent(container.NewCenter(content))
}

// Company Selection Screen
func (g *GUI) showCompanySelection() {
	title := widget.NewLabel("Welcome to OpenERP")
	title.TextStyle.Bold = true

	// Database path entry
	dbPathEntry := widget.NewEntry()
	dbPathEntry.SetPlaceHolder("Enter database path (or leave empty for erp.db)")
	dbPathEntry.SetText("erp.db")

	// Open/Create database button
	openBtn := widget.NewButton("Open Database", func() {
		dbPath := dbPathEntry.Text
		if dbPath == "" {
			dbPath = "erp.db"
		}

		db := &Database{}
		if err := db.OpenDatabase(dbPath); err != nil {
			dialog.ShowError(fmt.Errorf("Failed to open database: %w", err), g.window)
			return
		}

		g.db = db
		g.showCompanyList()
	})

	content := container.NewVBox(
		widget.NewLabel(""),
		title,
		widget.NewLabel(""),
		widget.NewLabel("Database Configuration"),
		dbPathEntry,
		openBtn,
	)

	g.window.SetContent(container.NewCenter(content))
}

// Company List Screen
func (g *GUI) showCompanyList() {
	companies, err := g.db.ListCompanies()
	if err != nil {
		dialog.ShowError(err, g.window)
		return
	}

	title := widget.NewLabel("Select or Create Company")
	title.TextStyle.Bold = true

	// Company list
	var companyList *widget.List
	companyList = widget.NewList(
		func() int { return len(companies) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Company Name")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(companies[id])
		},
	)

	companyList.OnSelected = func(id widget.ListItemID) {
		g.company = companies[id]
		if err := g.db.EnterCompany(g.company); err != nil {
			dialog.ShowError(err, g.window)
			return
		}
		g.showMainMenu()
	}

	// Create new company
	newCompanyEntry := widget.NewEntry()
	newCompanyEntry.SetPlaceHolder("New company name")

	createBtn := widget.NewButton("Create New Company", func() {
		companyName := strings.TrimSpace(newCompanyEntry.Text)
		if companyName == "" {
			dialog.ShowError(fmt.Errorf("Company name cannot be empty"), g.window)
			return
		}

		if err := g.db.CreateCompany(companyName); err != nil {
			dialog.ShowError(err, g.window)
			return
		}

		// Refresh list
		companies, _ = g.db.ListCompanies()
		companyList.Refresh()
		newCompanyEntry.SetText("")
		dialog.ShowInformation("Success", fmt.Sprintf("Company '%s' created", companyName), g.window)
	})

	content := container.NewBorder(
		container.NewVBox(title, widget.NewSeparator()),
		container.NewVBox(
			widget.NewSeparator(),
			widget.NewLabel("Create New Company:"),
			newCompanyEntry,
			createBtn,
		),
		nil,
		nil,
		companyList,
	)

	g.window.SetContent(content)
}

// Main Menu
func (g *GUI) showMainMenu() {
	title := widget.NewLabel(fmt.Sprintf("OpenERP - Company: %s", g.company))
	title.TextStyle.Bold = true

	info := widget.NewLabel("Tables are automatically created when a company is initialized.")
	info.Wrapping = fyne.TextWrapWord

	placeholder := widget.NewLabel("Data entry forms coming soon...")
	placeholder.Importance = widget.LowImportance

	changeCompanyBtn := widget.NewButton("Change Company", func() {
		g.showCompanyList()
	})

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		widget.NewLabel(""),
		info,
		widget.NewLabel(""),
		placeholder,
		widget.NewLabel(""),
		widget.NewSeparator(),
		changeCompanyBtn,
	)

	g.window.SetContent(container.NewCenter(content))
}
func formatValue(value interface{}) string {
	if value == nil {
		return "<nil>"
	}

	switch v := value.(type) {
	case []byte:
		return string(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func convertValue(value string, fieldType string) (interface{}, error) {
	if value == "" {
		return nil, nil
	}

	switch fieldType {
	case "Text", "Date":
		return value, nil
	case "Boolean":
		lower := strings.ToLower(value)
		if lower == "true" || lower == "1" || lower == "yes" {
			return 1, nil
		}
		return 0, nil
	case "Integer":
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, err
		}
		return i, nil
	case "Decimal":
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		return f, nil
	default:
		return value, nil
	}
}
