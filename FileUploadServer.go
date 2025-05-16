package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	uploadDirBase = "data_tmp" // Base directory for all uploads
)

// sha256Hash generates SHA-256 hash for a message
func sha256Hash(message string) string {
	hash := sha256.Sum256([]byte(message))
	return hex.EncodeToString(hash[:])
}

// checkSHA256 checks if input is a valid SHA-256 hash, returns the input if valid or computes the hash if not
func checkSHA256(input string) string {
	sha256Regex := regexp.MustCompile(`^[a-f0-9]{64}$`)
	if sha256Regex.MatchString(input) {
		return input // Input is a valid SHA256 hash
	}
	return sha256Hash(input) // Input is not a valid SHA256 hash, return its hash
}

// performSearch handles search form submission
func performSearch(w http.ResponseWriter, r *http.Request) {
	searchInput := strings.TrimSpace(r.URL.Query().Get("search-input"))
	if searchInput == "" {
		return
	}

	// Check if input is already a valid SHA-256 hash (64 hex characters)
	isValidHash, _ := regexp.MatchString(`^[a-fA-F0-9]{64}$`, searchInput)

	var hash string
	if isValidHash {
		// If input is already a valid hash, use it directly
		hash = searchInput
	} else {
		// Otherwise generate SHA-256 hash of the input
		hash = sha256Hash(searchInput)
	}

	// Check if the file exists
	if _, err := os.Stat(filepath.Join(uploadDirBase, hash, "index.html")); err == nil {
		// Redirect to the page
		http.Redirect(w, r, filepath.Join(uploadDirBase, hash, "index.html"), http.StatusFound)
		return
	}

	// File doesn't exist
	fmt.Fprintf(w, "File don't exists!")
}

func main() {
	// Create upload directory if it doesn't exist
	if _, err := os.Stat(uploadDirBase); os.IsNotExist(err) {
		os.MkdirAll(uploadDirBase, 0755)
	}

	// Set up file server for the data_tmp directory
	fs := http.FileServer(http.Dir("./"))
	http.Handle("/data_tmp/", fs)

	// Handle file upload and form submission - only at the root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only process requests for the exact root path
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		
		// Handle search query
		if r.Method == http.MethodGet && r.URL.Query().Get("search-input") != "" {
			performSearch(w, r)
			return
		}

		reply := r.URL.Query().Get("reply")

		// Handle form submission
		if r.Method == http.MethodPost {
			// Check if category was provided
			category := r.FormValue("category")
			if category == "" {
				// No further processing if category is missing
				renderTemplate(w, reply, "Please enter a category.")
				return
			}

			var fileContent []byte
			var originalFileName string
			var fileExtension = "txt" // Default extension for text content
			var isTextContent = false // Flag to track if content is from text area

			// Check if a file was uploaded
			file, header, err := r.FormFile("uploaded_file")
			if err == nil {
				defer file.Close()

				fileContent, err = ioutil.ReadAll(file)
				if err != nil {
					http.Error(w, "Error reading uploaded file", http.StatusInternalServerError)
					return
				}
				originalFileName = header.Filename
				fileExtension = strings.TrimPrefix(filepath.Ext(originalFileName), ".")
				isTextContent = false
			} else {
				// If no file uploaded, check for text content
				textContent := r.FormValue("text_content")
				if textContent != "" {
					fileContent = []byte(textContent)
					date := time.Now().Format("2006.01.02 15:04:05") // Just for naming purposes in index.html

					originalFileName = sha256Hash(textContent)

					fileContentLen := len(textContent)
					if fileContentLen > 50 {
						originalFileName = template.HTMLEscapeString(textContent[:50]) + " (" + date + ")"
					} else {
						originalFileName = template.HTMLEscapeString(textContent) + " (" + date + ")"
					}

					isTextContent = true
				}
			}

			if len(fileContent) > 0 {
				// Check if PHP file
				if strings.ToLower(fileExtension) == "php" {
					http.Error(w, "Error: PHP files are not allowed!", http.StatusBadRequest)
					return
				}

				// Check if category is the same as text content
				if category == r.FormValue("text_content") {
					http.Error(w, "Error: Category can't be the same of text contents.", http.StatusBadRequest)
					return
				}

				// Calculate SHA256 hashes
				fileHash := sha256Hash(string(fileContent))
				categoryHash := checkSHA256(category)

				// Determine file extension
				fileNameWithExtension := fileHash + "." + fileExtension

				// Construct directory paths
				fileUploadDir := filepath.Join(uploadDirBase, fileHash) // Folder name is file hash
				categoryDir := filepath.Join(uploadDirBase, categoryHash) // Folder name is category hash

				// Create directories if they don't exist
				if _, err := os.Stat(fileUploadDir); os.IsNotExist(err) {
					os.MkdirAll(fileUploadDir, 0755)
				}
				if _, err := os.Stat(categoryDir); os.IsNotExist(err) {
					os.MkdirAll(categoryDir, 0755)
				}

				// Save the content (either uploaded file or text content)
				destinationFilePath := filepath.Join(fileUploadDir, fileNameWithExtension)

				if _, err := os.Stat(destinationFilePath); err == nil {
					http.Error(w, "Error: File already exists!", http.StatusBadRequest)
					return
				}

				var saveSuccess bool
				if isTextContent {
					// Save text content directly
					err := ioutil.WriteFile(destinationFilePath, fileContent, 0644)
					saveSuccess = (err == nil)
				} else {
					// For uploaded files, create a new file and copy the content
					destFile, err := os.Create(destinationFilePath)
					if err != nil {
						http.Error(w, "Error creating destination file", http.StatusInternalServerError)
						return
					}
					defer destFile.Close()
					
					// Re-open the uploaded file since we already read it
					file, _, err = r.FormFile("uploaded_file")
					if err != nil {
						http.Error(w, "Error accessing uploaded file", http.StatusInternalServerError)
						return
					}
					defer file.Close()
					
					_, err = io.Copy(destFile, file)
					saveSuccess = (err == nil)
				}

				if !saveSuccess {
					http.Error(w, "Error saving content.", http.StatusInternalServerError)
					return
				}

				// Create empty file in category folder with hash + extension name
				categoryFilePath := filepath.Join(categoryDir, fileNameWithExtension)
				emptyFile, err := os.Create(categoryFilePath)
				if err != nil {
					http.Error(w, "Error creating empty file in category folder.", http.StatusInternalServerError)
					return
				}
				emptyFile.Close()

				contentHead := "<link rel='stylesheet' href='../../default.css'><script src='../../default.js'></script><script src='../../ads.js'></script><div id='ads' name='ads' class='ads'></div><div id='default' name='default' class='default'></div>"

				// Handle index.html inside file hash folder (for content links)
				indexPathFileFolder := filepath.Join(fileUploadDir, "index.html")
				if _, err := os.Stat(indexPathFileFolder); os.IsNotExist(err) {
					// Create index.html if it doesn't exist
					err = ioutil.WriteFile(indexPathFileFolder, []byte(contentHead), 0644)
					if err != nil {
						http.Error(w, "Error creating index file.", http.StatusInternalServerError)
						return
					}
				}

				linkReply := "<a href=\"../../?reply=" + template.HTMLEscapeString(fileHash) + "\">" + "[ Reply ]" + "</a> "
				linkToHash := linkReply + "<a href=\"../" + template.HTMLEscapeString(fileHash) + "/index.html\">" + "[ Open ]" + "</a> "
				linkToFileFolderIndex := linkToHash + "<a href=\"" + template.HTMLEscapeString(fileNameWithExtension) + "\">" + template.HTMLEscapeString(originalFileName) + "</a><br>"

				indexContentFileFolder, err := ioutil.ReadFile(indexPathFileFolder)
				if err != nil {
					http.Error(w, "Error reading index file.", http.StatusInternalServerError)
					return
				}

				if !strings.Contains(string(indexContentFileFolder), linkToFileFolderIndex) {
					// Append the new link to the index file
					indexFile, err := os.OpenFile(indexPathFileFolder, os.O_WRONLY, 0644)
					if err != nil {
						http.Error(w, "Error opening index file for writing.", http.StatusInternalServerError)
						return
					}
					defer indexFile.Close()
					_, err = indexFile.WriteString(string(indexContentFileFolder) + linkToFileFolderIndex)
					if err != nil {
						http.Error(w, "Error writing to index file.", http.StatusInternalServerError)
						return
					}
				}

				// Handle index.html inside category folder (for link to original content)
				indexPathCategoryFolder := filepath.Join(categoryDir, "index.html")
				if _, err := os.Stat(indexPathCategoryFolder); os.IsNotExist(err) {
					// Create index.html if it doesn't exist
					err = ioutil.WriteFile(indexPathCategoryFolder, []byte(contentHead), 0644)
					if err != nil {
						http.Error(w, "Error creating category index file.", http.StatusInternalServerError)
						return
					}
				}

				// Construct relative path to the content in the content hash folder
				relativePathToFile := "../" + fileHash + "/" + fileNameWithExtension

				categoryReply := "<a href=\"../../?reply=" + template.HTMLEscapeString(fileHash) + "\">" + "[ Reply ]" + "</a> "
				linkToHashCategory := categoryReply + "<a href=\"../" + template.HTMLEscapeString(fileHash) + "/index.html\">" + "[ Open ]" + "</a> "
				linkToCategoryFolderIndex := linkToHashCategory + "<a href=\"" + template.HTMLEscapeString(relativePathToFile) + "\">" + template.HTMLEscapeString(originalFileName) + "</a><br>"

				indexContentCategoryFolder, err := ioutil.ReadFile(indexPathCategoryFolder)
				if err != nil {
					http.Error(w, "Error reading category index file.", http.StatusInternalServerError)
					return
				}

				if !strings.Contains(string(indexContentCategoryFolder), linkToCategoryFolderIndex) {
					// Append the new link to the category index file
					categoryIndexFile, err := os.OpenFile(indexPathCategoryFolder, os.O_WRONLY, 0644)
					if err != nil {
						http.Error(w, "Error opening category index file for writing.", http.StatusInternalServerError)
						return
					}
					defer categoryIndexFile.Close()
					_, err = categoryIndexFile.WriteString(string(indexContentCategoryFolder) + linkToCategoryFolderIndex)
					if err != nil {
						http.Error(w, "Error writing to category index file.", http.StatusInternalServerError)
						return
					}
				}

				// Render success message and form
				fmt.Fprintf(w, "<p class='success'>Content processed successfully!</p>")
				fmt.Fprintf(w, "<p>Content saved in: <pre><a href='%s'>%s</a></pre></p>", 
					template.HTMLEscapeString(indexPathCategoryFolder), 
					template.HTMLEscapeString(indexPathCategoryFolder))
				
				renderTemplate(w, reply, "")
				return
			} else {
				renderTemplate(w, reply, "Please select a file or enter text content and provide a category.")
				return
			}
		}

		// Render the HTML form
		renderTemplate(w, reply, "")
	})

	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}

func renderTemplate(w http.ResponseWriter, reply string, errorMsg string) {
	html := `<!DOCTYPE html>
<html>
<head>
<title>File/Text Upload with Category</title>
</head>
<body>

<form method="GET" action="" id="search-form">
    <input type="text" id="search" name="search-input" placeholder="Enter file hash or category" required>
    <button type="submit">Search</button>
</form>

<h2>Upload File</h2>

<form action="/`

	if reply != "" {
		html += "?reply=" + template.HTMLEscapeString(reply)
	}

	html += `" method="post" enctype="multipart/form-data">
    <label for="uploaded_file">Select File:</label>
    <input type="file" name="uploaded_file" id="uploaded_file"><br><br>

    <label for="text_content">Or enter text content:</label><br>
    <textarea name="text_content" id="text_content" rows="5" cols="40"></textarea><br><br>

    <label for="category">Category:</label>
    <input type="text" name="category" id="category" value="`

	if reply != "" {
		html += template.HTMLEscapeString(reply)
	}

	html += `" required `

	if reply != "" {
		html += "readonly"
	}

	html += `><br><br>

    <input type="submit" value="Upload">
</form>`

	if errorMsg != "" {
		html += fmt.Sprintf("<p class='error'>%s</p>", template.HTMLEscapeString(errorMsg))
	}

	html += `
</body>
</html>`

	fmt.Fprint(w, html)
}