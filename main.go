package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"metamorph/pkg/llm"
	"os"
	"strings"
	"sync"
	"text/template"
)

func main() {

	inputCSV := flag.String("input", "", "Path to the input CSV file")
	promptTemplate := flag.String("prompt", "", "Prompt template for the LLM")
	outputCSV := flag.String("output", "", "Path to the output CSV file")
	numberOfWorkers := flag.Int("workers", 5, "Number of workers to use")

	flag.Parse()

	workChan := make(chan work)
	rowLocker := sync.Mutex{}
	wg := sync.WaitGroup{}

	ctx := context.Background()
	model, err := llm.NewVertex(ctx, "kodespaces")
	if err != nil {
		log.Fatal(err)
	}

	// Read prompt from file
	if *promptTemplate == "" {
		log.Fatal("Please provide a prompt template")
	}

	templ, err := os.ReadFile(*promptTemplate)
	if err != nil {
		log.Fatal(err)
	}

	// Prepare the prompt template
	tmpl, err := template.New("prompt").Parse(string(templ))
	if err != nil {
		log.Fatalf("Failed to parse prompt template: %v", err)
	}

	// Open the input CSV
	inFile, err := os.Open(*inputCSV)
	if err != nil {
		log.Fatalf("Failed to open input CSV file: %v", err)
	}
	defer inFile.Close()

	// Create a new CSV reader
	reader := csv.NewReader(inFile)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read CSV data: %v", err)
	}
	if len(records) == 0 {
		log.Fatal("Input CSV is empty")
	}

	// We'll treat the first row as a header row
	header := records[0]
	dataRows := records[1:]

	// Add a new column to the header for the LLM response
	header = append(header, "LLMResponse")

	for range *numberOfWorkers {
		go doWork(ctx, workChan, dataRows, tmpl, &rowLocker, model, &wg)
	}

	for i, row := range dataRows {
		wg.Add(1)
		workChan <- work{recordIndex: int32(i), rowData: row}
	}

	wg.Wait()
	close(workChan)

	// Write the output CSV
	outFile, err := os.Create(*outputCSV)
	if err != nil {
		log.Fatalf("Failed to create output CSV file: %v", err)
	}
	defer outFile.Close()

	writer := csv.NewWriter(outFile)
	defer writer.Flush()

	// Write the header row
	if err := writer.Write(header); err != nil {
		log.Fatalf("Failed to write header row: %v", err)
	}

	// Write each data row
	for _, row := range dataRows {
		if err := writer.Write(row); err != nil {
			log.Fatalf("Failed to write row: %v", err)
		}
	}

	fmt.Printf("Successfully wrote updated CSV with LLM responses to %s\n", *outputCSV)
}

func doWork(ctx context.Context, workChan chan work, dataRows [][]string, tmpl *template.Template, rowLocker *sync.Mutex, model *llm.VertexLLM, wg *sync.WaitGroup) {
	for rowToProcess := range workChan {
		data := make(map[string]string)
		for j, value := range rowToProcess.rowData {
			// "col1" will be row[0], "col2" => row[1], etc.
			key := fmt.Sprintf("col%d", j+1)
			data[key] = value
		}

		// Render the prompt for this row
		var promptBuilder strings.Builder
		if err := tmpl.Execute(&promptBuilder, data); err != nil {
			log.Printf("Error executing template for row %d: %v\n", rowToProcess.recordIndex, err)
			rowLocker.Lock()
			dataRows[rowToProcess.recordIndex] = append(dataRows[rowToProcess.recordIndex], "ERROR: "+err.Error())
			rowLocker.Unlock()
			wg.Done()
			continue
		}
		prompt := promptBuilder.String()

		response, err := model.Generate(ctx, prompt)
		if err != nil {
			log.Printf("LLM call failed for row %d: %v\n", rowToProcess.recordIndex, err)
			rowLocker.Lock()
			dataRows[rowToProcess.recordIndex] = append(dataRows[rowToProcess.recordIndex], "ERROR: "+err.Error())
			rowLocker.Unlock()
			wg.Done()
			continue
		}

		// Append the LLM response to the row
		rowLocker.Lock()
		dataRows[rowToProcess.recordIndex] = append(dataRows[rowToProcess.recordIndex], response)
		rowLocker.Unlock()

		log.Printf("LLM response for row %d: %s\n", rowToProcess.recordIndex, response)

		wg.Done()
	}
}

type work struct {
	recordIndex int32
	rowData     []string
}
