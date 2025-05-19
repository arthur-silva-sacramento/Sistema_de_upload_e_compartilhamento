<?php
    function sha256($message) {
        return hash('sha256', $message);
    }

    // Function to handle search form submission
    function performSearch() {
        if (isset($_GET['search-input'])) {
            $searchInput = trim($_GET['search-input']);

            if (empty($searchInput)) return;

            // Check if input is already a valid SHA-256 hash (64 hex characters)
            $isValidHash = preg_match('/^[a-fA-F0-9]{64}$/', $searchInput);

            if ($isValidHash) {
                // If input is already a valid hash, use it directly
                $hash = $searchInput;
            } else {
                // Otherwise generate SHA-256 hash of the input
                $hash = sha256($searchInput);
            }

            if (file_exists("data_tmp/$hash/index.html")) {
                // Redirect to the page instead of including it
                header("Location: data_tmp/$hash/index.html");
                exit(); // Important to prevent further script execution
            } else {
                echo "File don't exists!"; die;
            }
        }
    }

    // Call the function when form is submitted
    if ($_SERVER['REQUEST_METHOD'] === 'GET' && isset($_GET['search-input'])) {
        performSearch();
    }

    function check_sha256(string $input): string {
        $sha256_regex = '/^[a-f0-9]{64}$/'; // Regex for a 64-character hexadecimal string

        if (preg_match($sha256_regex, $input)) {
            return $input; // Input is a valid SHA256 hash
        } else {
            return hash('sha256', $input); // Input is not a valid SHA256 hash, return its hash
        }
    }

    if (isset($_GET['reply'])) {
        $reply = $_GET['reply'];
    } else {
        $reply = "";
    }

    if ($_SERVER['REQUEST_METHOD'] === 'POST') {
        // Configuration - Customize these paths and settings as needed
        $uploadDirBase = 'data_tmp'; // Base directory for all uploads
        $ownersDir = 'owners'; // Directory for BTC information
        $metadataDir = 'metadata'; // Directory for metadata JSON files

        // Create directories if they don't exist
        if (!is_dir($uploadDirBase)) {
            mkdir($uploadDirBase, 0777, true);
        }
        if (!is_dir($ownersDir)) {
            mkdir($ownersDir, 0777, true);
        }
        if (!is_dir($metadataDir)) {
            mkdir($metadataDir, 0777, true);
        }

        // Check if category was provided
        if (isset($_POST['category']) && !empty($_POST['category'])) {
            $categoryText = $_POST['category'];
            $fileContent = null;
            $originalFileName = null;
            $fileExtension = 'txt'; // Default extension for text content
            $isTextContent = false; // Flag to track if content is from text area

            // Check if a file was uploaded
            if (isset($_FILES['uploaded_file']) && $_FILES['uploaded_file']['error'] === UPLOAD_ERR_OK) {
                $uploadedFile = $_FILES['uploaded_file'];
                $fileContent = file_get_contents($uploadedFile['tmp_name']);
                $originalFileName = $uploadedFile['name'];
                $fileExtension = pathinfo($originalFileName, PATHINFO_EXTENSION);
                $isTextContent = false;
            } elseif (isset($_POST['text_content']) && !empty($_POST['text_content'])) {
                // If no file uploaded, check for text content
                $fileContent = $_POST['text_content'];
                $date = date("Y.m.d H:i:s"); // Just for naming purposes in index.html
                
                $originalFileName = hash('sha256', $fileContent);
                $fileContentLen = strlen($fileContent);

                if ($fileContentLen > 50) {
                    $originalFileName = htmlspecialchars(substr($fileContent, 0, 50)) . " ($date)"; 
                } else {
                    $originalFileName = htmlspecialchars(substr($fileContent, 0, 50)) . " ($date)";
                }

                $isTextContent = true;
            }

            if ($fileContent !== null) { // Proceed if we have either file content or text content
                if (strtolower($fileExtension) === 'php') {
                    die('Error: PHP files are not allowed!');
                }

                if ($_POST['category'] == $_POST['text_content']) {
                    die ("Error: Category can't be the same of text contents.");
                }

                // Calculate SHA256 hashes
                $fileHash = hash('sha256', $fileContent);
                $categoryHash = check_sha256($categoryText);

                // Determine file extension (already done above, default is 'txt' for text content)
                $fileNameWithExtension = $fileHash . '.' . $fileExtension; // Hash + extension as filename

                // Construct directory paths
                $fileUploadDir = $uploadDirBase . '/' . $fileHash; // Folder name is file hash
                $categoryDir = $uploadDirBase . '/' . $categoryHash; // Folder name is category hash

                // Create directories if they don't exist
                if (!is_dir($fileUploadDir)) {
                    mkdir($fileUploadDir, 0777, true); // Create file hash folder
                }
                if (!is_dir($categoryDir)) {
                    mkdir($categoryDir, 0777, true); // Create category hash folder
                }

                // Save the content (either uploaded file or text content)
                $destinationFilePath = $fileUploadDir . '/' . $fileNameWithExtension;

                if (file_exists($destinationFilePath)) {
                    die('Error: File already exists!');
                }

                if ($isTextContent) {

                    $file = fopen($destinationFilePath, 'w');
                    fwrite($file, $fileContent);
                    fclose($file);                                          

                    if ($saveResult !== false) {
                        $fileSaved = true;
                    } else {
                        $fileSaved = false;
                    }
                } else {
                    $fileSaved = move_uploaded_file($uploadedFile['tmp_name'], $destinationFilePath); // Save uploaded file
                }

                if ($fileSaved) {
                    // Content saved successfully

                    // Save BTC information if provided
                    if (isset($_POST['btc']) && !empty($_POST['btc'])) {
                        $btcFilePath = $ownersDir . '/' . $fileHash;
                        if (!file_exists($btcFilePath)) {                            
                            $file = fopen($btcFilePath, 'w');
                            fwrite($file, $_POST['btc']);
                            fclose($file);                          
                        }
                    }

                    // Save metadata if all required fields are provided
                    if (isset($_POST['user']) && !empty($_POST['user']) &&
                        isset($_POST['title']) && !empty($_POST['title']) &&
                        isset($_POST['description']) && !empty($_POST['description']) &&
                        isset($_POST['url']) && !empty($_POST['url'])) {
                        
                        $metadata = [
                            'user' => $_POST['user'],
                            'title' => $_POST['title'],
                            'description' => $_POST['description'],
                            'url' => $_POST['url']
                        ];
                        
                        $metadataFilePath = $metadataDir . '/' . $fileHash . '.json';
                        if (!file_exists($metadataFilePath)) {                            
                            $file = fopen($metadataFilePath, 'w');
                            fwrite($file, json_encode($metadata, JSON_PRETTY_PRINT));
                            fclose($file);
                        }
                    }

                    // Create empty file in category folder with hash + extension name
                    $categoryFilePath = $categoryDir . '/' . $fileNameWithExtension; // Empty file name is file hash + extension inside category folder
                    if (touch($categoryFilePath)) {
                        // Empty file created successfully
                        
                        $contentHead = "<link rel='stylesheet' href='../../default.css'><script src='../../default.js'></script><script src='../../ads.js'></script><div id='ads' name='ads' class='ads'></div><div id='default' name='default' class='default'></div>";
 
                        // Handle index.html inside file hash folder (for content links)
                        $indexPathFileFolder = $fileUploadDir . '/index.html';
                        if (!file_exists($indexPathFileFolder)) {
                            $file = fopen($indexPathFileFolder, 'a');
                            fwrite($file, $contentHead);
                            fclose($file);
                        }
  
                        $linkReply = '<a href="../../index.php?reply=' . htmlspecialchars($fileHash) . '">' . "[ Reply ]" . '</a> ';
                        $linkToHash = $linkReply . '<a href="../' . htmlspecialchars($fileHash) . '/index.html">' . "[ Open ]" . '</a> ';
                        $linkToFileFolderIndex = $linkToHash . '<a href="' . htmlspecialchars($fileNameWithExtension) . '">' . htmlspecialchars($originalFileName) . '</a><br>'; //Use original file name or 'text_content.txt' for link text
                        $indexContentFileFolder = file_get_contents($indexPathFileFolder);
                        if (strpos($indexContentFileFolder, $linkToFileFolderIndex) === false) {
                            $file = fopen($indexPathFileFolder, 'w');
                            fwrite($file, $indexContentFileFolder . $linkToFileFolderIndex);
                            fclose($file);  
                        }

                        // Handle index.html inside category folder (for link to original content)
                        $indexPathCategoryFolder = $categoryDir . '/index.html';
                        if (!file_exists($indexPathCategoryFolder)) {
                            $file = fopen($indexPathCategoryFolder, 'a');
                            fwrite($file, $contentHead);
                            fclose($file);                 
                        }

                        // Construct relative path to the content in the content hash folder
                        $relativePathToFile = '../' . $fileHash . '/' . $fileNameWithExtension;

                        $categoryReply = '<a href="../../index.php?reply=' . htmlspecialchars($fileHash) . '">' . "[ Reply ]" . '</a> ';
                        $linkToHashCategory = $categoryReply . '<a href="../' . htmlspecialchars($fileHash) . '/index.html">' . "[ Open ]" . '</a> ';
                        $linkToCategoryFolderIndex = $linkToHashCategory . '<a href="' . htmlspecialchars($relativePathToFile) . '">' . htmlspecialchars($originalFileName) . '</a><br>'; //Use original file name or 'text_content.txt' for link text
                        $indexContentCategoryFolder = file_get_contents($indexPathCategoryFolder);
                        if (strpos($indexContentCategoryFolder, $linkToCategoryFolderIndex) === false) {
                            $file = fopen($indexPathCategoryFolder, 'w');
                            fwrite($file, $indexContentCategoryFolder . $linkToCategoryFolderIndex);
                            fclose($file);  
                        }
                    }

                    echo "<p class='success'>Content processed successfully!</p>";
                    echo "<p>Content saved in: <pre><a href='" . htmlspecialchars($indexPathCategoryFolder) . "'>$indexPathCategoryFolder</a></pre></p>";
                } else {
                    echo "<p class='error'>Error creating empty file in category folder.</p>";
                }
            } else {
                echo "<p class='error'>Error saving content.</p>";
            }
        } else {
            echo "<p class='error'>Please select a file or enter text content and provide a category.</p>";
        }
    }
?>

<!DOCTYPE html>
<html>
<head>
    <title>File/Text Upload with Category</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }

        body {
            max-width: 600px;
            margin: 40px auto;
            padding: 0 20px;
            color: #333;
            line-height: 1.6;
        }

        h2 {
            font-size: 1.5em;
            margin-bottom: 20px;
            color: #222;
        }

        form {
            display: flex;
            flex-direction: column;
            gap: 15px;
        }

        label {
            font-size: 0.9em;
            color: #444;
        }

        input[type="text"],
        input[type="url"],
        textarea,
        input[type="file"] {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 5px;
            font-size: 1em;
            transition: border-color 0.2s;
        }

        input[type="text"]:focus,
        input[type="url"]:focus,
        textarea:focus {
            outline: none;
            border-color: #4a90e2;
        }

        textarea {
            resize: vertical;
            min-height: 100px;
        }

        button, input[type="submit"] {
            padding: 10px 20px;
            background: #4a90e2;
            color: white;
            border: none;
            cc
            border-radius: 5px;
            font-size: 1em;
            cursor: pointer;
            transition: background 0.2s;
        }

        button:hover, input[type="submit"]:hover {
            background: #357abd;
        }

        .search-form {
            flex-direction: row;
            gap: 10px;
        }

        .search-form input {
            flex: 1;
        }

        .more-options-link {
            color: #4a90e2;
            font-size: 0.9em;
            cursor: pointer;
            text-decoration: none;
            margin: 10px 0;
            display: inline-block;
        }

        .more-options-link:hover {
            text-decoration: underline;
        }

        .optional-fields {
            display: none;
            padding: 15px;
            background: #f8f9fa;
            border-radius: 5px;
        }

        .success {
            color: #2ecc71;
            font-size: 0.9em;
        }

        .error {
            color: #e74c3c;
            font-size: 0.9em;
        }
    </style>
    <script>
        document.addEventListener('DOMContentLoaded', () => {
            const moreOptionsLink = document.getElementById('more-options-link');
            const optionalFields = document.getElementById('optional-fields');

            moreOptionsLink.addEventListener('click', () => {
                const isHidden = optionalFields.style.display === 'none' || optionalFields.style.display === '';
                optionalFields.style.display = isHidden ? 'block' : 'none';
                moreOptionsLink.textContent = isHidden ? 'Less options' : 'More options';
            });
        });
    </script>
</head>
<body>
    <form method="GET" action="" id="search-form" class="search-form">
        <input type="text" id="search" name="search-input" placeholder="Enter file hash or category" required>
        <button type="submit">Search</button>
    </form>

    <h2>Upload File</h2>

    <form action="<?php echo htmlspecialchars($_SERVER['PHP_SELF']); if(isset($_GET['reply'])){echo "?reply=" . $_GET['reply'];} ?>" method="post" enctype="multipart/form-data">
        <label for="uploaded_file">Select File:</label>
        <input type="file" name="uploaded_file" id="uploaded_file">

        <label for="text_content">Or enter text content:</label>
        <textarea name="text_content" id="text_content" rows="5"></textarea>

        <label for="category">Category:</label>
        <input type="text" name="category" id="category" value="<?php if(isset($_GET['reply'])){echo $_GET['reply'];} ?>" required <?php if(isset($_GET['reply'])){echo "readonly";} ?>>

        <a id="more-options-link" class="more-options-link">More options</a>
        
        <div id="optional-fields" class="optional-fields">
            <label for="btc">BTC/PIX (optional):</label>
            <input type="text" name="btc" id="btc" placeholder="BTC address">
            
            <label for="user">User (optional):</label>
            <input type="text" name="user" id="user" placeholder="Username">
            
            <label for="title">Title (optional):</label>
            <input type="text" name="title" id="title" placeholder="Content title">
            
            <label for="description">Description (optional):</label>
            <input type="text" name="description" id="description" placeholder="Content description">
            
            <label for="url">URL (optional):</label>
            <input type="url" name="url" id="url" placeholder="Related URL">
        </div>

        <input type="submit" value="Upload">
    </form>
</body>
</html>
