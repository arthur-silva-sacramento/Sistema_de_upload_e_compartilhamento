import java.io.*;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpHandler;
import com.sun.net.httpserver.HttpServer;

public class FileUploadServer {
    private static final String UPLOAD_DIR_BASE = "data_tmp"; // Base directory for all uploads
    private static final Pattern SHA256_REGEX = Pattern.compile("^[a-f0-9]{64}$");
    private static final int PORT = 8080;

    public static void main(String[] args) throws Exception {
        // Create upload directory if it doesn't exist
        File uploadDir = new File(UPLOAD_DIR_BASE);
        if (!uploadDir.exists()) {
            uploadDir.mkdirs();
        }

        // Create HTTP server
        HttpServer server = HttpServer.create(new InetSocketAddress(PORT), 0);
        
        // Set up handler for root path
        server.createContext("/", new RootHandler());
        
        // Set up file server for the data_tmp directory
        server.createContext("/data_tmp/", new FileServerHandler());
        
        server.setExecutor(null); // Use default executor
        server.start();
        
        System.out.println("Server started at :" + PORT);
    }

    /**
     * Generates SHA-256 hash for a message
     */
    private static String sha256Hash(String message) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(message.getBytes(StandardCharsets.UTF_8));
            StringBuilder hexString = new StringBuilder();
            
            for (byte b : hash) {
                String hex = Integer.toHexString(0xff & b);
                if (hex.length() == 1) {
                    hexString.append('0');
                }
                hexString.append(hex);
            }
            
            return hexString.toString();
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException("SHA-256 algorithm not found.", e);
        }
    }

    /**
     * Generates SHA-256 hash for binary data
     */
    private static String sha256Hash(byte[] data) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(data);
            StringBuilder hexString = new StringBuilder();
            
            for (byte b : hash) {
                String hex = Integer.toHexString(0xff & b);
                if (hex.length() == 1) {
                    hexString.append('0');
                }
                hexString.append(hex);
            }
            
            return hexString.toString();
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException("SHA-256 algorithm not found.", e);
        }
    }

    /**
     * Checks if input is a valid SHA-256 hash, returns the input if valid or computes the hash if not
     */
    private static String checkSHA256(String input) {
        Matcher matcher = SHA256_REGEX.matcher(input);
        if (matcher.matches()) {
            return input; // Input is a valid SHA256 hash
        }
        return sha256Hash(input); // Input is not a valid SHA256 hash, return its hash
    }

    /**
     * Performs search based on input (direct hash or hashed input)
     */
    private static void performSearch(HttpExchange exchange) throws IOException {
        // Parse query parameters
        String query = exchange.getRequestURI().getQuery();
        Map<String, String> params = parseQueryParams(query);
        String searchInput = params.get("search-input");
        
        if (searchInput == null || searchInput.trim().isEmpty()) {
            sendResponse(exchange, 200, "");
            return;
        }
        
        searchInput = searchInput.trim();
        
        // Check if input is already a valid SHA-256 hash (64 hex characters)
        boolean isValidHash = SHA256_REGEX.matcher(searchInput).matches();
        
        String hash;
        if (isValidHash) {
            // If input is already a valid hash, use it directly
            hash = searchInput;
        } else {
            // Otherwise generate SHA-256 hash of the input
            hash = sha256Hash(searchInput);
        }
        
        // Check if the file exists
        File indexFile = new File(UPLOAD_DIR_BASE + File.separator + hash + File.separator + "index.html");
        if (indexFile.exists()) {
            // Redirect to the page
            exchange.getResponseHeaders().set("Location", UPLOAD_DIR_BASE + "/" + hash + "/index.html");
            exchange.sendResponseHeaders(302, -1);
            exchange.close();
            return;
        }
        
        // File doesn't exist
        sendResponse(exchange, 200, "File don't exists!");
    }

    /**
     * Parse query parameters from the URI
     */
    private static Map<String, String> parseQueryParams(String query) {
        Map<String, String> result = new HashMap<>();
        if (query == null || query.isEmpty()) {
            return result;
        }
        
        String[] pairs = query.split("&");
        for (String pair : pairs) {
            int idx = pair.indexOf("=");
            if (idx > 0) {
                String key = pair.substring(0, idx);
                String value = idx < pair.length() - 1 ? pair.substring(idx + 1) : "";
                result.put(key, value);
            }
        }
        
        return result;
    }

    /**
     * Send HTTP response with given status code and content
     */
    private static void sendResponse(HttpExchange exchange, int statusCode, String content) throws IOException {
        byte[] responseBytes = content.getBytes(StandardCharsets.UTF_8);
        exchange.sendResponseHeaders(statusCode, responseBytes.length);
        try (OutputStream os = exchange.getResponseBody()) {
            os.write(responseBytes);
        }
    }

    /**
     * HTML-escapes a string
     */
    private static String htmlEscape(String input) {
        return input.replace("&", "&amp;")
                   .replace("<", "&lt;")
                   .replace(">", "&gt;")
                   .replace("\"", "&quot;")
                   .replace("'", "&#39;");
    }

    /**
     * Handler for the root path
     */
    static class RootHandler implements HttpHandler {
        @Override
        public void handle(HttpExchange exchange) throws IOException {
            String path = exchange.getRequestURI().getPath();
            
            // Only process requests for the exact root path
            if (!"/".equals(path)) {
                sendResponse(exchange, 404, "Not Found");
                return;
            }
            
            String method = exchange.getRequestMethod();
            
            // Handle search query for GET requests
            if ("GET".equals(method)) {
                String query = exchange.getRequestURI().getQuery();
                if (query != null && query.contains("search-input=")) {
                    performSearch(exchange);
                    return;
                }
            }
            
            // Extract reply parameter if present
            String reply = "";
            String query = exchange.getRequestURI().getQuery();
            if (query != null && query.startsWith("reply=")) {
                reply = query.substring(6);
            }
            
            // Handle form submission
            if ("POST".equals(method)) {
                handleFormSubmission(exchange, reply);
                return;
            }
            
            // Render the HTML form for GET requests
            renderTemplate(exchange, reply, "");
        }
        
        /**
         * Handle form submission
         */
        private void handleFormSubmission(HttpExchange exchange, String reply) throws IOException {
            // Parse the multipart form data
            MultipartFormData formData = parseMultipartFormData(exchange);
            
            // Check if category was provided
            String category = formData.getFormFields().get("category");
            if (category == null || category.isEmpty()) {
                // No further processing if category is missing
                renderTemplate(exchange, reply, "Please enter a category.");
                return;
            }
            
            byte[] fileContent = null;
            String originalFileName = "";
            String fileExtension = "txt"; // Default extension for text content
            boolean isTextContent = false; // Flag to track if content is from text area
            
            // Check if a file was uploaded
            if (formData.getFileContent() != null && formData.getFileContent().length > 0) {
                fileContent = formData.getFileContent();
                originalFileName = formData.getFileName();
                if (originalFileName != null && !originalFileName.isEmpty()) {
                    int dotIndex = originalFileName.lastIndexOf('.');
                    if (dotIndex > 0) {
                        fileExtension = originalFileName.substring(dotIndex + 1);
                    }
                }
                isTextContent = false;
            } else {
                // If no file uploaded, check for text content
                String textContent = formData.getFormFields().get("text_content");
                if (textContent != null && !textContent.isEmpty()) {
                    fileContent = textContent.getBytes(StandardCharsets.UTF_8);
                    SimpleDateFormat dateFormat = new SimpleDateFormat("yyyy.MM.dd HH:mm:ss");
                    String date = dateFormat.format(new Date()); // Just for naming purposes in index.html
                    
                    originalFileName = sha256Hash(textContent);
                    
                    int fileContentLen = textContent.length();
                    if (fileContentLen > 50) {
                        originalFileName = htmlEscape(textContent.substring(0, 50)) + " (" + date + ")";
                    } else {
                        originalFileName = htmlEscape(textContent) + " (" + date + ")";
                    }
                    
                    isTextContent = true;
                }
            }
            
            if (fileContent != null && fileContent.length > 0) {
                // Check if PHP file
                if ("php".equalsIgnoreCase(fileExtension)) {
                    sendResponse(exchange, 400, "Error: PHP files are not allowed!");
                    return;
                }
                
                // Check if category is the same as text content
                String textContent = formData.getFormFields().get("text_content");
                if (category.equals(textContent)) {
                    sendResponse(exchange, 400, "Error: Category can't be the same of text contents.");
                    return;
                }
                
                // Calculate SHA256 hashes
                String fileHash = isTextContent ? 
                    sha256Hash(new String(fileContent, StandardCharsets.UTF_8)) : 
                    sha256Hash(fileContent);
                String categoryHash = checkSHA256(category);
                
                // Determine file extension
                String fileNameWithExtension = fileHash + "." + fileExtension;
                
                // Construct directory paths
                String fileUploadDir = UPLOAD_DIR_BASE + File.separator + fileHash; // Folder name is file hash
                String categoryDir = UPLOAD_DIR_BASE + File.separator + categoryHash; // Folder name is category hash
                
                // Create directories if they don't exist
                new File(fileUploadDir).mkdirs();
                new File(categoryDir).mkdirs();
                
                // Save the content (either uploaded file or text content)
                String destinationFilePath = fileUploadDir + File.separator + fileNameWithExtension;
                
                File destinationFile = new File(destinationFilePath);
                if (destinationFile.exists()) {
                    sendResponse(exchange, 400, "Error: File already exists!");
                    return;
                }
                
                boolean saveSuccess = false;
                try {
                    // Save the content
                    try (FileOutputStream fos = new FileOutputStream(destinationFilePath)) {
                        fos.write(fileContent);
                    }
                    saveSuccess = true;
                } catch (IOException e) {
                    e.printStackTrace();
                }
                
                if (!saveSuccess) {
                    sendResponse(exchange, 500, "Error saving content.");
                    return;
                }
                
                // Create empty file in category folder with hash + extension name
                String categoryFilePath = categoryDir + File.separator + fileNameWithExtension;
                try {
                    new File(categoryFilePath).createNewFile();
                } catch (IOException e) {
                    sendResponse(exchange, 500, "Error creating empty file in category folder.");
                    return;
                }
                
                String contentHead = "<link rel='stylesheet' href='../../default.css'><script src='../../default.js'></script><script src='../../ads.js'></script><div id='ads' name='ads' class='ads'></div><div id='default' name='default' class='default'></div>";
                
                // Handle index.html inside file hash folder (for content links)
                String indexPathFileFolder = fileUploadDir + File.separator + "index.html";
                File indexFileFolder = new File(indexPathFileFolder);
                
                if (!indexFileFolder.exists()) {
                    // Create index.html if it doesn't exist
                    try (FileWriter writer = new FileWriter(indexFileFolder)) {
                        writer.write(contentHead);
                    } catch (IOException e) {
                        sendResponse(exchange, 500, "Error creating index file.");
                        return;
                    }
                }
                
                String linkReply = "<a href=\"../../?reply=" + htmlEscape(fileHash) + "\">" + "[ Reply ]" + "</a> ";
                String linkToHash = linkReply + "<a href=\"../" + htmlEscape(fileHash) + "/index.html\">" + "[ Open ]" + "</a> ";
                String linkToFileFolderIndex = linkToHash + "<a href=\"" + htmlEscape(fileNameWithExtension) + "\">" + htmlEscape(originalFileName) + "</a><br>";
                
                // Read current index file content
                String indexContentFileFolder = "";
                try {
                    indexContentFileFolder = new String(Files.readAllBytes(Paths.get(indexPathFileFolder)), StandardCharsets.UTF_8);
                } catch (IOException e) {
                    sendResponse(exchange, 500, "Error reading index file.");
                    return;
                }
                
                if (!indexContentFileFolder.contains(linkToFileFolderIndex)) {
                    // Append the new link to the index file
                    try (FileWriter writer = new FileWriter(indexFileFolder, true)) {
                        writer.write(linkToFileFolderIndex);
                    } catch (IOException e) {
                        sendResponse(exchange, 500, "Error writing to index file.");
                        return;
                    }
                }
                
                // Handle index.html inside category folder (for link to original content)
                String indexPathCategoryFolder = categoryDir + File.separator + "index.html";
                File indexCategoryFolder = new File(indexPathCategoryFolder);
                
                if (!indexCategoryFolder.exists()) {
                    // Create index.html if it doesn't exist
                    try (FileWriter writer = new FileWriter(indexCategoryFolder)) {
                        writer.write(contentHead);
                    } catch (IOException e) {
                        sendResponse(exchange, 500, "Error creating category index file.");
                        return;
                    }
                }
                
                // Construct relative path to the content in the content hash folder
                String relativePathToFile = "../" + fileHash + "/" + fileNameWithExtension;
                
                String categoryReply = "<a href=\"../../?reply=" + htmlEscape(fileHash) + "\">" + "[ Reply ]" + "</a> ";
                String linkToHashCategory = categoryReply + "<a href=\"../" + htmlEscape(fileHash) + "/index.html\">" + "[ Open ]" + "</a> ";
                String linkToCategoryFolderIndex = linkToHashCategory + "<a href=\"" + htmlEscape(relativePathToFile) + "\">" + htmlEscape(originalFileName) + "</a><br>";
                
                // Read current category index file content
                String indexContentCategoryFolder = "";
                try {
                    indexContentCategoryFolder = new String(Files.readAllBytes(Paths.get(indexPathCategoryFolder)), StandardCharsets.UTF_8);
                } catch (IOException e) {
                    sendResponse(exchange, 500, "Error reading category index file.");
                    return;
                }
                
                if (!indexContentCategoryFolder.contains(linkToCategoryFolderIndex)) {
                    // Append the new link to the category index file
                    try (FileWriter writer = new FileWriter(indexCategoryFolder, true)) {
                        writer.write(linkToCategoryFolderIndex);
                    } catch (IOException e) {
                        sendResponse(exchange, 500, "Error writing to category index file.");
                        return;
                    }
                }
                
                // Render success message and form
                StringBuilder response = new StringBuilder();
                response.append("<p class='success'>Content processed successfully!</p>");
                response.append("<p>Content saved in: <pre><a href='").append(htmlEscape(indexPathCategoryFolder))
                       .append("'>").append(htmlEscape(indexPathCategoryFolder)).append("</a></pre></p>");
                
                // Add the form template
                response.append(getTemplateHtml(reply, ""));
                
                sendResponse(exchange, 200, response.toString());
                return;
            } else {
                renderTemplate(exchange, reply, "Please select a file or enter text content and provide a category.");
                return;
            }
        }
        
        /**
         * Parse multipart form data from HTTP request
         */
        private MultipartFormData parseMultipartFormData(HttpExchange exchange) throws IOException {
            MultipartFormData result = new MultipartFormData();
            Map<String, String> formFields = new HashMap<>();
            result.setFormFields(formFields);
            
            // Check if it's a multipart form
            String contentType = exchange.getRequestHeaders().getFirst("Content-Type");
            if (contentType == null || !contentType.startsWith("multipart/form-data")) {
                // Parse as regular form data
                try (BufferedReader reader = new BufferedReader(new InputStreamReader(exchange.getRequestBody()))) {
                    String formData = reader.readLine();
                    if (formData != null) {
                        String[] pairs = formData.split("&");
                        for (String pair : pairs) {
                            int idx = pair.indexOf("=");
                            if (idx > 0) {
                                String key = pair.substring(0, idx);
                                String value = idx < pair.length() - 1 ? pair.substring(idx + 1) : "";
                                // URL decode the value
                                value = java.net.URLDecoder.decode(value, StandardCharsets.UTF_8.name());
                                formFields.put(key, value);
                            }
                        }
                    }
                }
                return result;
            }
            
            // Get the boundary from the content type
            String boundary = "";
            int boundaryIndex = contentType.indexOf("boundary=");
            if (boundaryIndex != -1) {
                boundary = contentType.substring(boundaryIndex + 9);
                if (boundary.startsWith("\"") && boundary.endsWith("\"")) {
                    boundary = boundary.substring(1, boundary.length() - 1);
                }
            }
            
            if (boundary.isEmpty()) {
                return result;
            }
            
            // Read all bytes from request body
            ByteArrayOutputStream buffer = new ByteArrayOutputStream();
            try (InputStream is = exchange.getRequestBody()) {
                byte[] data = new byte[8192];
                int bytesRead;
                while ((bytesRead = is.read(data, 0, data.length)) != -1) {
                    buffer.write(data, 0, bytesRead);
                }
            }
            byte[] requestBody = buffer.toByteArray();
            
            // Convert the boundary to bytes for binary comparison
            byte[] boundaryBytes = ("--" + boundary).getBytes(StandardCharsets.UTF_8);
            
            // Find all boundary positions
            int pos = 0;
            while (pos < requestBody.length) {
                int boundaryPos = indexOf(requestBody, boundaryBytes, pos);
                if (boundaryPos == -1) {
                    break;
                }
                
                // Find the end of this part (next boundary or end of data)
                int nextBoundaryPos = indexOf(requestBody, boundaryBytes, boundaryPos + boundaryBytes.length);
                if (nextBoundaryPos == -1) {
                    break;
                }
                
                // Extract this part
                byte[] partBytes = new byte[nextBoundaryPos - boundaryPos - boundaryBytes.length];
                System.arraycopy(requestBody, boundaryPos + boundaryBytes.length, partBytes, 0, partBytes.length);
                
                // Find headers and content
                int headerEnd = indexOf(partBytes, "\r\n\r\n".getBytes(StandardCharsets.UTF_8), 0);
                if (headerEnd == -1) {
                    pos = nextBoundaryPos;
                    continue;
                }
                
                // Parse headers
                byte[] headerBytes = new byte[headerEnd];
                System.arraycopy(partBytes, 0, headerBytes, 0, headerEnd);
                String headers = new String(headerBytes, StandardCharsets.UTF_8);
                
                // Extract the field name from the header
                String fieldName = "";
                Pattern namePattern = Pattern.compile("name=\"([^\"]+)\"");
                Matcher nameMatcher = namePattern.matcher(headers);
                if (nameMatcher.find()) {
                    fieldName = nameMatcher.group(1);
                }
                
                // Extract content (skip the 4 bytes of \r\n\r\n)
                int contentStart = headerEnd + 4;
                int contentLength = partBytes.length - contentStart;
                
                // Check if this part is a file
                if (headers.contains("filename=")) {
                    Pattern filenamePattern = Pattern.compile("filename=\"([^\"]+)\"");
                    Matcher filenameMatcher = filenamePattern.matcher(headers);
                    if (filenameMatcher.find()) {
                        String filename = filenameMatcher.group(1);
                        result.setFileName(filename);
                        
                        // Extract file content
                        byte[] fileContent = new byte[contentLength];
                        System.arraycopy(partBytes, contentStart, fileContent, 0, contentLength);
                        result.setFileContent(fileContent);
                    }
                } else {
                    // This is a regular form field
                    byte[] contentBytes = new byte[contentLength];
                    System.arraycopy(partBytes, contentStart, contentBytes, 0, contentLength);
                    String content = new String(contentBytes, StandardCharsets.UTF_8).trim();
                    formFields.put(fieldName, content);
                }
                
                pos = nextBoundaryPos;
            }
            
            return result;
        }
        
        /**
         * Helper method to find byte array within another byte array
         */
        private int indexOf(byte[] array, byte[] target, int fromIndex) {
            if (target.length == 0) {
                return fromIndex;
            }
            
            outer:
            for (int i = fromIndex; i < array.length - target.length + 1; i++) {
                for (int j = 0; j < target.length; j++) {
                    if (array[i + j] != target[j]) {
                        continue outer;
                    }
                }
                return i;
            }
            return -1;
        }
        
        /**
         * Render the HTML template with optional error message
         */
        private void renderTemplate(HttpExchange exchange, String reply, String errorMsg) throws IOException {
            String html = getTemplateHtml(reply, errorMsg);
            sendResponse(exchange, 200, html);
        }
        
        /**
         * Get the HTML template
         */
        private String getTemplateHtml(String reply, String errorMsg) {
            StringBuilder html = new StringBuilder();
            html.append("<!DOCTYPE html>\n")
                .append("<html>\n")
                .append("<head>\n")
                .append("<title>File/Text Upload with Category</title>\n")
                .append("</head>\n")
                .append("<body>\n\n")
                .append("<form method=\"GET\" action=\"\" id=\"search-form\">\n")
                .append("    <input type=\"text\" id=\"search\" name=\"search-input\" placeholder=\"Enter file hash or category\" required>\n")
                .append("    <button type=\"submit\">Search</button>\n")
                .append("</form>\n\n")
                .append("<h2>Upload File</h2>\n\n")
                .append("<form action=\"/");
            
            if (reply != null && !reply.isEmpty()) {
                html.append("?reply=").append(htmlEscape(reply));
            }
            
            html.append("\" method=\"post\" enctype=\"multipart/form-data\">\n")
                .append("    <label for=\"uploaded_file\">Select File:</label>\n")
                .append("    <input type=\"file\" name=\"uploaded_file\" id=\"uploaded_file\"><br><br>\n\n")
                .append("    <label for=\"text_content\">Or enter text content:</label><br>\n")
                .append("    <textarea name=\"text_content\" id=\"text_content\" rows=\"5\" cols=\"40\"></textarea><br><br>\n\n")
                .append("    <label for=\"category\">Category:</label>\n")
                .append("    <input type=\"text\" name=\"category\" id=\"category\" value=\"");
            
            if (reply != null && !reply.isEmpty()) {
                html.append(htmlEscape(reply));
            }
            
            html.append("\" required ");
            
            if (reply != null && !reply.isEmpty()) {
                html.append("readonly");
            }
            
            html.append("><br><br>\n\n")
                .append("    <input type=\"submit\" value=\"Upload\">\n")
                .append("</form>");
            
            if (errorMsg != null && !errorMsg.isEmpty()) {
                html.append(String.format("<p class='error'>%s</p>", htmlEscape(errorMsg)));
            }
            
            html.append("\n</body>\n</html>");
            
            return html.toString();
        }
    }
    
    /**
     * Handler for serving static files from the data_tmp directory
     */
    static class FileServerHandler implements HttpHandler {
        @Override
        public void handle(HttpExchange exchange) throws IOException {
            String requestPath = exchange.getRequestURI().getPath();
            
            // Convert URL path to file system path
            String filePath = "." + requestPath;
            File file = new File(filePath);
            
            if (!file.exists() || file.isDirectory()) {
                sendResponse(exchange, 404, "File not found");
                return;
            }
            
            // Set content type based on file extension
            String contentType = getContentType(filePath);
            exchange.getResponseHeaders().set("Content-Type", contentType);
            
            // Send the file
            byte[] fileData = Files.readAllBytes(Paths.get(filePath));
            exchange.sendResponseHeaders(200, fileData.length);
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(fileData);
            }
        }
        
        /**
         * Determine content type based on file extension
         */
        private String getContentType(String filePath) {
            if (filePath.endsWith(".html")) {
                return "text/html";
            } else if (filePath.endsWith(".css")) {
                return "text/css";
            } else if (filePath.endsWith(".js")) {
                return "application/javascript";
            } else if (filePath.endsWith(".txt")) {
                return "text/plain";
            } else {
                return "application/octet-stream";
            }
        }
    }
    
    /**
     * Class to hold multipart form data
     */
    static class MultipartFormData {
        private Map<String, String> formFields;
        private String fileName;
        private byte[] fileContent;
        
        public Map<String, String> getFormFields() {
            return formFields;
        }
        
        public void setFormFields(Map<String, String> formFields) {
            this.formFields = formFields;
        }
        
        public String getFileName() {
            return fileName;
        }
        
        public void setFileName(String fileName) {
            this.fileName = fileName;
        }
        
        public byte[] getFileContent() {
            return fileContent;
        }
        
        public void setFileContent(byte[] fileContent) {
            this.fileContent = fileContent;
        }
    }
}