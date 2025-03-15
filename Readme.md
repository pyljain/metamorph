# Metamorph

This project processes rows from a CSV file, generates prompts for a Large Language Model (LLM), and appends the LLM’s responses back into an output CSV. It uses Google Cloud Vertex AI (via `llm.NewVertex`) as the backend for generating responses, though you can adapt it for other LLM services.

## Features

- Reads a CSV file (first row is treated as header, subsequent rows are data).
- Uses a template file to dynamically build the prompt for each row based on its columns.
- Calls an LLM for each row and appends the response to the row’s data.
- Outputs the updated CSV to a specified file.
- Processes multiple rows in parallel, configurable via a worker count.

## Project Structure

- **main.go**  
  - Parses command-line flags (paths to CSV files, template, worker count).
  - Reads data from the input CSV and stores it in memory.
  - Spawns worker goroutines to generate prompts, call the LLM, and store responses.
  - Writes the updated data (including LLM responses) to an output CSV.

- **pkg/llm**  
  - Contains logic to instantiate and call the LLM (in this case, Vertex AI).

- **prompt template file**  
  - A file containing the text/template syntax that is filled with row data (referenced via `{{.col1}}`, `{{.col2}}`, etc.).  

## Getting Started

### Prerequisites

1. Go 1.19+ (or compatible version).
2. A working environment configured to call Vertex AI (or your chosen LLM). For Vertex AI, ensure:
   - You have the correct environment variables set or default application credentials available (e.g., `GOOGLE_APPLICATION_CREDENTIALS`).
   - The project ID or region is set appropriately in `llm.NewVertex`.

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/metamorph.git
   cd metamorph
   ```
2. Install dependencies (if any) using:
   ```bash
   go mod download
   ```

### Usage

1. Create a **prompt template**. This is a text file with placeholders in Go’s template format. For example:
   ```txt
   Please summarize the following text:
   {{.col2}}
   ```
   In this example, `col2` would refer to the second column of a row in your CSV.

2. Prepare your **input CSV**. For example:
   ```csv
   id,text
   1,"This is a short text that needs summarizing."
   2,"Another piece of text."
   ```

3. Run the program with required flags:
   ```bash
   go run main.go \
       -input=/path/to/input.csv \
       -prompt=/path/to/prompt_template.txt \
       -output=/path/to/output.csv \
       -workers=5
   ```
   - `-input` – Path to the input CSV.
   - `-prompt` – Path to the file containing your prompt template.
   - `-output` – Path to the output CSV.
   - `-workers` – Number of goroutines that process CSV rows in parallel.

### How It Works

1. **Reading CSV**  
   The application reads the CSV rows into memory. The first row is a header row; the subsequent rows contain data.

2. **Prompt Generation**  
   For each row, the program constructs a map like `{"col1": row[0], "col2": row[1], ...}`. It then uses the Go `text/template` engine to fill in the prompt template with row-specific data.

3. **LLM Call**  
   The constructed prompt is sent to the LLM (`model.Generate(ctx, prompt)`). The response is captured.

4. **Storing the Response**  
   The LLM response is appended to the corresponding row in memory.

5. **Output CSV**  
   After processing all rows, the updated records are written to the output CSV with an extra column for the LLM’s responses.

### Concurrency Model

- A channel (`workChan`) distributes rows to worker goroutines.
- Each worker calls the LLM and appends the response to the row.
- A `sync.Mutex` protects shared memory access to rows.
- A `sync.WaitGroup` waits for all workers to finish.

### Customizing

- **LLM Provider**  
  If you want to use a different LLM instead of Vertex AI, replace calls in `llm.NewVertex` with your chosen provider’s library calls.
- **Prompt Logic**  
  The prompt template can be made as simple or complex as you like. Use standard Go template features (`{{if ...}}`, `{{range ...}}`, etc.).
- **Error Handling**  
  Currently, errors for a row append an “ERROR:” string to that row. Adjust this logic as desired.

### Troubleshooting

- **Authentication / Credentials**  
  Check that your environment is set up for Vertex AI calls.  
- **Empty CSV**  
  The application exits if the input CSV is empty.
- **Template Errors**  
  If the prompt template references columns that don’t exist (e.g., `{{.col3}}` for a row that only has 2 columns), an error is logged and that row’s response will be set to an error message.