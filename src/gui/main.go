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
	"github.com/hansjlachmann/openerp-go-gui/types"
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

	objectDesignerBtn := widget.NewButton("Object Designer", func() {
		g.showObjectDesigner()
	})

	dataManagerBtn := widget.NewButton("Data Manager", func() {
		g.showDataManager()
	})

	changeCompanyBtn := widget.NewButton("Change Company", func() {
		g.showCompanyList()
	})

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		widget.NewLabel(""),
		objectDesignerBtn,
		dataManagerBtn,
		widget.NewLabel(""),
		widget.NewSeparator(),
		changeCompanyBtn,
	)

	g.window.SetContent(container.NewCenter(content))
}

// Object Designer Screen
func (g *GUI) showObjectDesigner() {
	title := widget.NewLabel("Object Designer")
	title.TextStyle.Bold = true

	createTableBtn := widget.NewButton("Create Table", func() {
		g.showCreateTable()
	})

	addFieldBtn := widget.NewButton("Add Field to Table", func() {
		g.showAddField()
	})

	listTablesBtn := widget.NewButton("List Tables", func() {
		g.showTableList()
	})

	deleteTableBtn := widget.NewButton("Delete Table", func() {
		g.showDeleteTable()
	})

	backBtn := widget.NewButton("Back to Main Menu", func() {
		g.showMainMenu()
	})

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		createTableBtn,
		addFieldBtn,
		listTablesBtn,
		deleteTableBtn,
		widget.NewSeparator(),
		backBtn,
	)

	g.window.SetContent(container.NewCenter(content))
}

// Create Table Dialog
func (g *GUI) showCreateTable() {
	tableNameEntry := widget.NewEntry()
	tableNameEntry.SetPlaceHolder("Enter table name (e.g., Customer)")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Table Name", Widget: tableNameEntry},
		},
		OnSubmit: func() {
			tableName := strings.TrimSpace(tableNameEntry.Text)
			if tableName == "" {
				dialog.ShowError(fmt.Errorf("Table name cannot be empty"), g.window)
				return
			}

			if err := g.db.CreateTable(tableName); err != nil {
				dialog.ShowError(err, g.window)
				return
			}

			dialog.ShowInformation("Success",
				fmt.Sprintf("Table '%s' created successfully", tableName),
				g.window)
			g.showObjectDesigner()
		},
		OnCancel: func() {
			g.showObjectDesigner()
		},
	}

	content := container.NewVBox(
		widget.NewLabel("Create New Table"),
		widget.NewSeparator(),
		form,
	)

	g.window.SetContent(container.NewCenter(content))
}

// Add Field Dialog
func (g *GUI) showAddField() {
	// Get tables
	tables, err := g.db.ListTables()
	if err != nil {
		dialog.ShowError(err, g.window)
		return
	}

	if len(tables) == 0 {
		dialog.ShowError(fmt.Errorf("No tables available. Create a table first."), g.window)
		g.showObjectDesigner()
		return
	}

	// Table selection
	tableSelect := widget.NewSelect(tables, nil)
	tableSelect.PlaceHolder = "Select a table"

	// Field name
	fieldNameEntry := widget.NewEntry()
	fieldNameEntry.SetPlaceHolder("Field name (e.g., name, email)")

	// Field type
	fieldTypeSelect := widget.NewSelect([]string{"Text", "Boolean", "Date", "Decimal", "Integer"}, nil)
	fieldTypeSelect.PlaceHolder = "Select field type"

	// Primary key checkbox
	isPKCheck := widget.NewCheck("Primary Key", nil)

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Table", Widget: tableSelect},
			{Text: "Field Name", Widget: fieldNameEntry},
			{Text: "Field Type", Widget: fieldTypeSelect},
			{Text: "", Widget: isPKCheck},
		},
		OnSubmit: func() {
			if tableSelect.Selected == "" {
				dialog.ShowError(fmt.Errorf("Please select a table"), g.window)
				return
			}
			if strings.TrimSpace(fieldNameEntry.Text) == "" {
				dialog.ShowError(fmt.Errorf("Field name cannot be empty"), g.window)
				return
			}
			if fieldTypeSelect.Selected == "" {
				dialog.ShowError(fmt.Errorf("Please select a field type"), g.window)
				return
			}

			err := g.db.AddField(
				tableSelect.Selected,
				strings.TrimSpace(fieldNameEntry.Text),
				fieldTypeSelect.Selected,
				isPKCheck.Checked,
			)
			if err != nil {
				dialog.ShowError(err, g.window)
				return
			}

			pkText := ""
			if isPKCheck.Checked {
				pkText = " [PRIMARY KEY]"
			}
			dialog.ShowInformation("Success",
				fmt.Sprintf("Field '%s' (%s)%s added to table '%s'",
					fieldNameEntry.Text, fieldTypeSelect.Selected, pkText, tableSelect.Selected),
				g.window)
			g.showObjectDesigner()
		},
		OnCancel: func() {
			g.showObjectDesigner()
		},
	}

	content := container.NewVBox(
		widget.NewLabel("Add Field to Table"),
		widget.NewSeparator(),
		form,
	)

	g.window.SetContent(container.NewCenter(content))
}

// Table List
func (g *GUI) showTableList() {
	tables, err := g.db.ListTables()
	if err != nil {
		dialog.ShowError(err, g.window)
		g.showObjectDesigner()
		return
	}

	title := widget.NewLabel(fmt.Sprintf("Tables in Company: %s", g.company))
	title.TextStyle.Bold = true

	if len(tables) == 0 {
		content := container.NewVBox(
			title,
			widget.NewSeparator(),
			widget.NewLabel("No tables found"),
			widget.NewButton("Back", func() { g.showObjectDesigner() }),
		)
		g.window.SetContent(container.NewCenter(content))
		return
	}

	// Create table list with details
	var tableWidgets []fyne.CanvasObject
	for _, table := range tables {
		tableLabel := widget.NewLabel(fmt.Sprintf("📋 %s (Full: %s$%s)", table, g.company, table))
		tableLabel.TextStyle.Bold = true

		// Get fields
		fields, err := g.db.ListFields(table)
		if err == nil && len(fields) > 0 {
			fieldsText := "Fields:\n"
			for _, field := range fields {
				pkMarker := ""
				if field.IsPrimaryKey {
					pkMarker = " 🔑"
				}
				fieldsText += fmt.Sprintf("  • %s (%s)%s\n", field.Name, field.Type, pkMarker)
			}
			fieldLabel := widget.NewLabel(fieldsText)
			tableWidgets = append(tableWidgets, tableLabel, fieldLabel)
		} else {
			tableWidgets = append(tableWidgets, tableLabel)
		}

		tableWidgets = append(tableWidgets, widget.NewSeparator())
	}

	backBtn := widget.NewButton("Back to Object Designer", func() {
		g.showObjectDesigner()
	})

	tableWidgets = append(tableWidgets, backBtn)

	content := container.NewVBox(tableWidgets...)
	scrollContainer := container.NewScroll(content)

	g.window.SetContent(container.NewBorder(
		container.NewVBox(title, widget.NewSeparator()),
		nil, nil, nil,
		scrollContainer,
	))
}

// Delete Table
func (g *GUI) showDeleteTable() {
	tables, err := g.db.ListTables()
	if err != nil {
		dialog.ShowError(err, g.window)
		g.showObjectDesigner()
		return
	}

	if len(tables) == 0 {
		dialog.ShowError(fmt.Errorf("No tables available"), g.window)
		g.showObjectDesigner()
		return
	}

	tableSelect := widget.NewSelect(tables, nil)
	tableSelect.PlaceHolder = "Select table to delete"

	deleteBtn := widget.NewButton("Delete Table", func() {
		if tableSelect.Selected == "" {
			dialog.ShowError(fmt.Errorf("Please select a table"), g.window)
			return
		}

		// Confirmation dialog
		dialog.ShowConfirm("Confirm Deletion",
			fmt.Sprintf("⚠️  WARNING: This will permanently delete table '%s$%s'!\n\nAre you sure?",
				g.company, tableSelect.Selected),
			func(confirmed bool) {
				if confirmed {
					if err := g.db.DeleteTable(tableSelect.Selected); err != nil {
						dialog.ShowError(err, g.window)
						return
					}
					dialog.ShowInformation("Success",
						fmt.Sprintf("Table '%s' deleted successfully", tableSelect.Selected),
						g.window)
					g.showObjectDesigner()
				}
			},
			g.window)
	})

	cancelBtn := widget.NewButton("Cancel", func() {
		g.showObjectDesigner()
	})

	content := container.NewVBox(
		widget.NewLabel("Delete Table"),
		widget.NewSeparator(),
		tableSelect,
		deleteBtn,
		cancelBtn,
	)

	g.window.SetContent(container.NewCenter(content))
}

// Data Manager Screen
func (g *GUI) showDataManager() {
	tables, err := g.db.ListTables()
	if err != nil {
		dialog.ShowError(err, g.window)
		g.showMainMenu()
		return
	}

	if len(tables) == 0 {
		dialog.ShowError(fmt.Errorf("No tables available. Create a table first."), g.window)
		g.showMainMenu()
		return
	}

	title := widget.NewLabel("Data Manager - Select Table")
	title.TextStyle.Bold = true

	tableList := widget.NewList(
		func() int { return len(tables) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Table Name")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(tables[id])
		},
	)

	tableList.OnSelected = func(id widget.ListItemID) {
		g.showTableDataManager(tables[id])
	}

	backBtn := widget.NewButton("Back to Main Menu", func() {
		g.showMainMenu()
	})

	content := container.NewBorder(
		container.NewVBox(title, widget.NewSeparator()),
		backBtn,
		nil, nil,
		tableList,
	)

	g.window.SetContent(content)
}

// Table Data Manager
func (g *GUI) showTableDataManager(tableName string) {
	title := widget.NewLabel(fmt.Sprintf("Data Manager - %s$%s", g.company, tableName))
	title.TextStyle.Bold = true

	viewAllBtn := widget.NewButton("View All Records", func() {
		g.showAllRecords(tableName)
	})

	addRecordBtn := widget.NewButton("Add New Record", func() {
		g.showAddRecord(tableName)
	})

	viewRecordBtn := widget.NewButton("View Single Record", func() {
		g.showViewRecord(tableName)
	})

	updateRecordBtn := widget.NewButton("Update Record", func() {
		g.showUpdateRecord(tableName)
	})

	deleteRecordBtn := widget.NewButton("Delete Record", func() {
		g.showDeleteRecord(tableName)
	})

	backBtn := widget.NewButton("Back to Table Selection", func() {
		g.showDataManager()
	})

	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		viewAllBtn,
		addRecordBtn,
		viewRecordBtn,
		updateRecordBtn,
		deleteRecordBtn,
		widget.NewSeparator(),
		backBtn,
	)

	g.window.SetContent(container.NewCenter(content))
}

// View All Records
func (g *GUI) showAllRecords(tableName string) {
	records, err := g.db.ListRecords(tableName)
	if err != nil {
		dialog.ShowError(err, g.window)
		return
	}

	fields, err := g.db.ListFields(tableName)
	if err != nil {
		dialog.ShowError(err, g.window)
		return
	}

	title := widget.NewLabel(fmt.Sprintf("All Records - %s", tableName))
	title.TextStyle.Bold = true

	if len(records) == 0 {
		content := container.NewVBox(
			title,
			widget.NewSeparator(),
			widget.NewLabel("No records found"),
			widget.NewButton("Back", func() { g.showTableDataManager(tableName) }),
		)
		g.window.SetContent(container.NewCenter(content))
		return
	}

	// Build records display
	var recordWidgets []fyne.CanvasObject
	for i, record := range records {
		recordLabel := widget.NewLabel(fmt.Sprintf("Record #%d:", i+1))
		recordLabel.TextStyle.Bold = true
		recordWidgets = append(recordWidgets, recordLabel)

		// Show PK fields first
		for _, field := range fields {
			if field.IsPrimaryKey {
				value := formatValue(record[field.Name])
				recordWidgets = append(recordWidgets,
					widget.NewLabel(fmt.Sprintf("  🔑 %s: %v", field.Name, value)))
			}
		}

		// Show created_at
		if createdAt, ok := record["created_at"]; ok {
			recordWidgets = append(recordWidgets,
				widget.NewLabel(fmt.Sprintf("  ⏰ created_at: %v", formatValue(createdAt))))
		}

		// Show regular fields
		for _, field := range fields {
			if !field.IsPrimaryKey {
				value := formatValue(record[field.Name])
				recordWidgets = append(recordWidgets,
					widget.NewLabel(fmt.Sprintf("  • %s: %v", field.Name, value)))
			}
		}

		recordWidgets = append(recordWidgets, widget.NewSeparator())
	}

	backBtn := widget.NewButton("Back", func() {
		g.showTableDataManager(tableName)
	})

	content := container.NewVBox(recordWidgets...)
	scrollContainer := container.NewScroll(content)

	g.window.SetContent(container.NewBorder(
		container.NewVBox(title, widget.NewSeparator()),
		backBtn,
		nil, nil,
		scrollContainer,
	))
}

// Add New Record
func (g *GUI) showAddRecord(tableName string) {
	fields, err := g.db.ListFields(tableName)
	if err != nil {
		dialog.ShowError(err, g.window)
		return
	}

	title := widget.NewLabel(fmt.Sprintf("Add New Record - %s", tableName))
	title.TextStyle.Bold = true

	// Create form items for each field
	entries := make(map[string]*widget.Entry)
	var formItems []*widget.FormItem

	for _, field := range fields {
		entry := widget.NewEntry()
		entry.SetPlaceHolder(fmt.Sprintf("Enter %s value", field.Type))
		entries[field.Name] = entry

		label := field.Name
		if field.IsPrimaryKey {
			label = field.Name + " 🔑"
		}
		formItems = append(formItems, &widget.FormItem{
			Text:   fmt.Sprintf("%s (%s)", label, field.Type),
			Widget: entry,
		})
	}

	form := &widget.Form{
		Items: formItems,
		OnSubmit: func() {
			record := make(map[string]interface{})
			for fieldName, entry := range entries {
				// Find field type
				var fieldType string
				for _, f := range fields {
					if f.Name == fieldName {
						fieldType = f.Type
						break
					}
				}

				value := strings.TrimSpace(entry.Text)
				converted, err := convertValue(value, fieldType)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Invalid value for %s: %v", fieldName, err), g.window)
					return
				}
				record[fieldName] = converted
			}

			_, err := g.db.InsertRecord(tableName, record)
			if err != nil {
				dialog.ShowError(err, g.window)
				return
			}

			dialog.ShowInformation("Success", "Record added successfully", g.window)
			g.showTableDataManager(tableName)
		},
		OnCancel: func() {
			g.showTableDataManager(tableName)
		},
	}

	content := container.NewVBox(title, widget.NewSeparator(), form)
	scrollContainer := container.NewScroll(content)
	g.window.SetContent(scrollContainer)
}

// View Single Record
func (g *GUI) showViewRecord(tableName string) {
	g.getPrimaryKeyAndExecute(tableName, "View Record", func(pk map[string]interface{}) {
		record, err := g.db.GetRecord(tableName, pk)
		if err != nil {
			dialog.ShowError(err, g.window)
			return
		}

		// Display record
		var recordText string
		for key, value := range record {
			recordText += fmt.Sprintf("%s: %v\n", key, formatValue(value))
		}

		dialog.ShowInformation("Record Details", recordText, g.window)
	})
}

// Update Record
func (g *GUI) showUpdateRecord(tableName string) {
	g.getPrimaryKeyAndExecute(tableName, "Update Record", func(pk map[string]interface{}) {
		// Get current record
		record, err := g.db.GetRecord(tableName, pk)
		if err != nil {
			dialog.ShowError(err, g.window)
			return
		}

		fields, _ := g.db.ListFields(tableName)

		// Create form for updates
		entries := make(map[string]*widget.Entry)
		var formItems []*widget.FormItem

		for _, field := range fields {
			if field.IsPrimaryKey {
				continue // Skip PK fields
			}

			entry := widget.NewEntry()
			currentValue := formatValue(record[field.Name])
			entry.SetPlaceHolder(fmt.Sprintf("Current: %v", currentValue))
			entries[field.Name] = entry

			formItems = append(formItems, &widget.FormItem{
				Text:   fmt.Sprintf("%s (%s)", field.Name, field.Type),
				Widget: entry,
			})
		}

		form := &widget.Form{
			Items: formItems,
			OnSubmit: func() {
				updates := make(map[string]interface{})
				for fieldName, entry := range entries {
					value := strings.TrimSpace(entry.Text)
					if value == "" {
						continue // Skip empty fields
					}

					// Find field type
					var fieldType string
					for _, f := range fields {
						if f.Name == fieldName {
							fieldType = f.Type
							break
						}
					}

					converted, err := convertValue(value, fieldType)
					if err != nil {
						dialog.ShowError(fmt.Errorf("Invalid value for %s: %v", fieldName, err), g.window)
						return
					}
					updates[fieldName] = converted
				}

				if len(updates) == 0 {
					dialog.ShowInformation("Info", "No changes made", g.window)
					g.showTableDataManager(tableName)
					return
				}

				err := g.db.UpdateRecord(tableName, pk, updates)
				if err != nil {
					dialog.ShowError(err, g.window)
					return
				}

				dialog.ShowInformation("Success", "Record updated successfully", g.window)
				g.showTableDataManager(tableName)
			},
			OnCancel: func() {
				g.showTableDataManager(tableName)
			},
		}

		title := widget.NewLabel("Update Record (leave blank to skip field)")
		content := container.NewVBox(title, widget.NewSeparator(), form)
		scrollContainer := container.NewScroll(content)
		g.window.SetContent(scrollContainer)
	})
}

// Delete Record
func (g *GUI) showDeleteRecord(tableName string) {
	g.getPrimaryKeyAndExecute(tableName, "Delete Record", func(pk map[string]interface{}) {
		// Get record to show what will be deleted
		record, err := g.db.GetRecord(tableName, pk)
		if err != nil {
			dialog.ShowError(err, g.window)
			return
		}

		var recordText string
		for key, value := range record {
			recordText += fmt.Sprintf("%s: %v\n", key, formatValue(value))
		}

		dialog.ShowConfirm("Confirm Deletion",
			fmt.Sprintf("⚠️  Are you sure you want to delete this record?\n\n%s", recordText),
			func(confirmed bool) {
				if confirmed {
					if err := g.db.DeleteRecord(tableName, pk); err != nil {
						dialog.ShowError(err, g.window)
						return
					}
					dialog.ShowInformation("Success", "Record deleted successfully", g.window)
					g.showTableDataManager(tableName)
				}
			},
			g.window)
	})
}

// Helper: Get primary key values from user
func (g *GUI) getPrimaryKeyAndExecute(tableName, title string, callback func(map[string]interface{})) {
	fields, err := g.db.ListFields(tableName)
	if err != nil {
		dialog.ShowError(err, g.window)
		return
	}

	// Get PK fields
	var pkFields []types.FieldInfo
	for _, field := range fields {
		if field.IsPrimaryKey {
			pkFields = append(pkFields, field)
		}
	}

	if len(pkFields) == 0 {
		dialog.ShowError(fmt.Errorf("No primary key fields defined"), g.window)
		return
	}

	// Create form for PK values
	entries := make(map[string]*widget.Entry)
	var formItems []*widget.FormItem

	for _, field := range pkFields {
		entry := widget.NewEntry()
		entry.SetPlaceHolder(fmt.Sprintf("Enter %s", field.Type))
		entries[field.Name] = entry

		formItems = append(formItems, &widget.FormItem{
			Text:   fmt.Sprintf("%s 🔑 (%s)", field.Name, field.Type),
			Widget: entry,
		})
	}

	form := &widget.Form{
		Items: formItems,
		OnSubmit: func() {
			pk := make(map[string]interface{})
			for fieldName, entry := range entries {
				// Find field type
				var fieldType string
				for _, f := range pkFields {
					if f.Name == fieldName {
						fieldType = f.Type
						break
					}
				}

				value := strings.TrimSpace(entry.Text)
				converted, err := convertValue(value, fieldType)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Invalid value for %s: %v", fieldName, err), g.window)
					return
				}
				pk[fieldName] = converted
			}

			callback(pk)
		},
		OnCancel: func() {
			g.showTableDataManager(tableName)
		},
	}

	titleLabel := widget.NewLabel(title + " - Enter Primary Key")
	content := container.NewVBox(titleLabel, widget.NewSeparator(), form)
	g.window.SetContent(container.NewCenter(content))
}

// Helper functions
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
