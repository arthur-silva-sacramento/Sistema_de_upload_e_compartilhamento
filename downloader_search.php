<?php
function searchJsonFiles($dir, $searchTerm) {
    $results = [];
    $items = new RecursiveIteratorIterator(
        new RecursiveDirectoryIterator($dir, RecursiveDirectoryIterator::SKIP_DOTS),
        RecursiveIteratorIterator::SELF_FIRST
    );
    
    foreach ($items as $item) {
        if ($item->isFile() && strtolower($item->getExtension()) === 'json') {
            $content = file_get_contents($item->getPathname());
            $json = json_decode($content, true);
            
            if (isset($json['title']) && stripos($json['title'], $searchTerm) !== false) {
                $results[] = [
                    'path' => $item->getPathname(),
                    'relativePath' => str_replace($dir . DIRECTORY_SEPARATOR, '', $item->getPathname()),
                    'title' => $json['title']
                ];
            }
        }
    }
    
    return $results;
}

$searchTerm = $_GET['search'] ?? '';
$results = [];
if (file_exists('categories_json') && is_dir('categories_json')) {
    $results = searchJsonFiles('categories_json', $searchTerm);
}
?>

<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>JSON File Search</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; }
        .search-box { margin-bottom: 20px; }
        .search-box input { padding: 8px; width: 300px; }
        .search-box button { padding: 8px 15px; }
        .results { margin-top: 20px; }
        .file-item { 
            margin: 10px 0; 
            padding: 10px; 
            border: 1px solid #ddd; 
            border-radius: 4px;
        }
        .file-path { color: #666; font-size: 0.9em; }
        a { color: #3498db; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .no-results { color: #666; font-style: italic; }
    </style>
</head>
<body>
    <h1>Search JSON Files</h1>
    
    <div class="search-box">
        <form method="get" action="">
            <input type="text" name="search" value="<?php echo htmlspecialchars($searchTerm); ?>" 
                   placeholder="Enter title to search...">
            <button type="submit">Search</button>
        </form>
    </div>
    
    <div class="results">
        <?php if (!empty($searchTerm)): ?>
            <h2>Results for "<?php echo htmlspecialchars($searchTerm); ?>"</h2>
            
            <?php if (!empty($results)): ?>
                <?php foreach ($results as $file): ?>
                    <div class="file-item">
                        <div class="file-title">
                            <strong><?php echo htmlspecialchars($file['title']); ?></strong>
                        </div>
                        <div class="file-path">
                            <a href="<?php echo htmlspecialchars($file['path']); ?>" target="_blank">
                                <?php echo htmlspecialchars($file['relativePath']); ?>
                            </a>
                        </div>
                    </div>
                <?php endforeach; ?>
            <?php else: ?>
                <p class="no-results">No matching files found.</p>
            <?php endif; ?>
        <?php elseif (empty($searchTerm) && !empty($_GET)): ?>
            <p class="no-results">Please enter a search term.</p>
        <?php endif; ?>
    </div>
</body>
</html>