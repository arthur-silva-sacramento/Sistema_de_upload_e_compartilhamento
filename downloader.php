<?php
// Ensure the url directory exists
if (!file_exists('url')) {
    mkdir('url', 0755, true);
}
// Ensure the other directory exists
if (!file_exists('other')) {
    mkdir('other', 0755, true);
}
// Function to sanitize input
function sanitizeInput($data) {
    return htmlspecialchars(strip_tags(trim($data)));
}
// Function to download URL content
function downloadUrl($url) {
    $ch = curl_init();
    curl_setopt($ch, CURLOPT_URL, $url);
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, 1);
    curl_setopt($ch, CURLOPT_FOLLOWLOCATION, 1);
    curl_setopt($ch, CURLOPT_USERAGENT, 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36');
    $content = curl_exec($ch);
    curl_close($ch);
    return $content;
}
// Function to get file extension from URL
function getExtensionFromUrl($url) {
    $path = parse_url($url, PHP_URL_PATH);
    if ($path) {
        $extension = pathinfo($path, PATHINFO_EXTENSION);
        if ($extension) {
            return strtolower($extension);
        }
    }
    return null;
}
// Process form submission
if ($_SERVER['REQUEST_METHOD'] === 'POST' && isset($_POST['url'])) {
    $url = sanitizeInput($_POST['url']);
    $user = isset($_POST['user']) ? sanitizeInput($_POST['user']) : '';
    $title = isset($_POST['title']) ? sanitizeInput($_POST['title']) : '';
    $description = isset($_POST['description']) ? sanitizeInput($_POST['description']) : '';
    $category = isset($_POST['category']) ? sanitizeInput($_POST['category']) : '';
    $btc = isset($_POST['btc']) ? sanitizeInput($_POST['btc']) : '';
    $pix = isset($_POST['pix']) ? sanitizeInput($_POST['pix']) : '';
    
    // Generate SHA256 hash of the URL
    $urlHash = hash('sha256', $url);
    
    // Save URL to text file using fopen
    $urlFile = fopen("url/{$urlHash}.txt", 'w');
    if ($urlFile) {
        fwrite($urlFile, $url);
        fclose($urlFile);
    }
    
    // Download URL content
    $content = downloadUrl($url);
    
    // Get file extension from URL
    $extension = getExtensionFromUrl($url);
    $currentDate = date('Ymd');
    
    // Determine save directory
    $saveDir = $extension ? $extension : 'other';

    $saveDirJSON = "categories_json" . "/" . $saveDir; 

    $saveDir = "categories" . "/" . $saveDir;
    
    if (!file_exists($saveDir)) {
        mkdir($saveDir, 0755, true);
    }
    
    // Create date directory
    $dateDir = "{$saveDir}/{$currentDate}";
    if (!file_exists($dateDir)) {
        mkdir($dateDir, 0755, true);
    }
    
    // Save downloaded content using fopen
    $filename = $extension ? "{$urlHash}.{$extension}" : $urlHash;
    $filePath = "{$dateDir}/{$filename}";
    $contentFile = fopen($filePath, 'w');
    if ($contentFile) {
        fwrite($contentFile, $content);
        fclose($contentFile);
    }
    
    // Save optional fields if any are provided
    if ($user || $title || $description || $category || $btc || $pix) {
        $metadata = [
            'url' => $url,
            'user' => $user,
            'title' => $title,
            'description' => $description,
            'category' => $category,
            'btc' => $btc,
            'pix' => $pix,
            'date' => $currentDate,
            'hash' => $urlHash
        ];
        
        // Create category directory if it doesn't exist
        $categoryDir = "{$saveDirJSON}/{$category}";
        if (!file_exists($categoryDir)) {
            mkdir($categoryDir, 0755, true);
        }
        
        // Create date directory inside category
        $categoryDateDir = "{$categoryDir}/{$currentDate}";
        if (!file_exists($categoryDateDir)) {
            mkdir($categoryDateDir, 0755, true);
        }
        
        // Save metadata as JSON using fopen
        $jsonFile = fopen("{$categoryDateDir}/{$urlHash}.json", 'w');
        if ($jsonFile) {
            fwrite($jsonFile, json_encode($metadata, JSON_PRETTY_PRINT));
            fclose($jsonFile);
        }
    }
    
    // Create the success message with a link to the downloaded file
    $relativeFilePath = str_replace($_SERVER['DOCUMENT_ROOT'], '', $filePath);
    $successMessage = '<a href="' . htmlspecialchars($relativeFilePath) . '" target="_blank">URL saved successfully!</a>';
    $successMessage =  $successMessage . ' <a href="' . htmlspecialchars("{$categoryDateDir}/{$urlHash}.json") . '" target="_blank">[view JSON]</a>';
    $result = ['success' => true, 'message' => $successMessage];
}
?>

<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>URL Downloader</title>
    <style>

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            background: linear-gradient(135deg, #34C759 0%, #FFFFFF 100%);
            line-height: 1.6;
            color: #1A1A1A;
        }
        
        .container {
            max-width: 900px;
            margin: auto;
            padding: 10px;
            flex-grow: 1;
            display: flex;
            flex-direction: column;
            justify-content: center;
        }        
        
        .url-input-group {
            display: flex;
            flex-direction: row;
            gap: 1rem;
            margin-bottom: 1.5rem;
            align-items: center;
        }
      
        .url-input-group input[type="text"] {
            flex-grow: 1;
            height: 60px;
            width: 90%;
            padding: 0 1.5rem;
            font-size: 1.125rem;
            border: none;
            border-radius: 5px;
            background: linear-gradient(135deg, #F5F5F5 0%, #E0E0E0 100%);
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
            transition: all 0.3s ease;
        }
        
        .url-input-group input[type="text"]:focus {
            outline: none;
            box-shadow: 0 0 0 4px rgba(52, 199, 89, 0.2);
        }
        
        .url-input-group input[type="submit"] {
            height: 60px;
            padding: 0 2rem;
            font-size: 1.125rem;
            background: linear-gradient(135deg, #34C759 0%, #28A745 100%);
            border: none;
            color: white;
            border-radius: 12px;
            cursor: pointer;
            transition: all 0.3s ease;
            font-weight: 600;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
            margin: 0;
            white-space: nowrap;
        }
        
        .url-input-group input[type="submit"]:hover {
            background: linear-gradient(135deg, #28A745 0%, #34C759 100%);
            transform: translateY(-2px);
            box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2);
        }
        
        .more-options-toggle {
            display: inline-block;
            margin: 1rem 0;
            color: #34C759;
            cursor: pointer;
            text-decoration: none;
            user-select: none;
            font-weight: 500;
            transition: color 0.3s ease;
            font-size: 1rem;
        }
        
        .more-options-toggle:hover {
            color: #28A745;
        }
        
        .more-options {
            display: none;
            padding-top: 1.5rem;
            margin-top: 1rem;
            border-top: 1px solid rgba(0, 0, 0, 0.1);
        }
        
        label {
            display: block;
            margin-top: 1.5rem;
            font-weight: 600;
            color: #1A1A1A;
            font-size: 1rem;
        }
        
        input[type="text"], textarea {
            width: 100%;
            padding: 0.75rem;
            margin-top: 0.5rem;
            border: none;
            border-radius: 8px;
            box-sizing: border-box;
            font-size: 1rem;
            background: #FFFFFF;
            box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
            transition: all 0.3s ease;
        }
        
        input[type="text"]:focus, textarea:focus {
            outline: none;
            box-shadow: 0 0 0 3px rgba(52, 199, 89, 0.2);
        }
        
        textarea {
            height: 120px;
            resize: vertical;
        }
        
        /* We're only keeping this for any other submit buttons in the form */
        input[type="submit"] {
            background: linear-gradient(135deg, #34C759 0%, #28A745 100%);
            color: white;
            padding: 0.75rem 1.5rem;
            border: none;
            border-radius: 12px;
            cursor: pointer;
            margin-top: 1.5rem;
            font-size: 1.125rem;
            font-weight: 600;
            transition: all 0.3s ease;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        }
        
        input[type="submit"]:hover {
            background: linear-gradient(135deg, #28A745 0%, #34C759 100%);
            transform: translateY(-2px);
            box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2);
        }
        
        .required:after {
            content: " *";
            color: #FF3B30;
        }
        
        .result {
            margin-top: 1.5rem;
            padding: 1rem;
            border-radius: 8px;
            font-size: 1rem;
        }
        
        .success {
            background-color: rgba(52, 199, 89, 0.1);
            color: #1A1A1A;
            border: 1px solid rgba(52, 199, 89, 0.3);
        }
        
        .error {
            background-color: rgba(255, 59, 48, 0.1);
            color: #1A1A1A;
            border: 1px solid rgba(255, 59, 48, 0.3);
        }
        
        footer {
            text-align: center;
            padding: 1.5rem;
            font-size: 0.875rem;
            color: #666;
            background: rgba(255, 255, 255, 0.9);
            border-top: 1px solid rgba(0, 0, 0, 0.05);
            margin-top: auto;
        }

.banner-container {
  grid-column: 1 / -1;
  text-align: center;
  margin: 20px 0;
}
        
.banner-ad {
  width: 100%;
  max-width: 728px;
  display: block;
  margin: 0 auto;
  border-radius: 8px;
}  
    </style>
    <script>
        function toggleMoreOptions() {
            var options = document.getElementById('more-options');
            var toggleText = document.getElementById('toggle-text');
            if (options.style.display === 'none' || options.style.display === '') {
                options.style.display = 'block';
                toggleText.innerText = 'Hide Options';
            } else {
                options.style.display = 'none';
                toggleText.innerText = 'More Options';
            }
        }
    </script>
</head>
<body>
    <div class="container">
        <h1>Downloader</h1>
        
        <div align="right"><a href="downloader_search.php">Search</a> <a href="downloader_pt.php">PT</a></div>

        <?php if (isset($result)): ?>
            <div class="result <?php echo $result['success'] ? 'success' : 'error'; ?>">
                <?php echo $result['message']; ?>
            </div>
        <?php endif; ?>
        
        <form method="post" action="">
            <div class="url-input-group">
                <input type="text" id="url" name="url" placeholder="Enter URL here (e.g., https://example.com/file.pdf)" required>
                <input type="submit" value="Download">
            </div>
            
            <div class="more-options-toggle" onclick="toggleMoreOptions()">
                <span id="toggle-text">More Options</span>
            </div>
            
            <div id="more-options" class="more-options">
                <label for="user">User</label>
                <input type="text" id="user" name="user" placeholder="Your username">
                
                <label for="title">Title</label>
                <input type="text" id="title" name="title" placeholder="Document title">
                
                <label for="description">Description</label>
                <textarea id="description" name="description" placeholder="Document description"></textarea>
                
                <label for="category">Category</label>
                <input type="text" id="category" name="category" placeholder="Default category will be used if empty">
                
                <label for="btc">BTC Address</label>
                <input type="text" id="btc" name="btc" placeholder="Bitcoin address">
                
                <label for="pix">PIX Key</label>
                <input type="text" id="pix" name="pix" placeholder="PIX key">
            </div>
        </form>
    </div>

<div class="banner-container">
  <a href="https://3gp.neocities.org/redirect.html" target="_blank" rel="noopener">
    <img src="https://3gp.neocities.org/banner.jpg" alt="Banner Ad" class="banner-ad">
  </a>
</div>
    
    <footer>
        <p>&copy; 2025 URL Downloader. All rights reserved.</p>
    </footer>
</body>
</html>