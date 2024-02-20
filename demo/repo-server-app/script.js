window.onload = function() {
    fetch('/files')
        .then(response => response.json())
        .then(filePaths => {
            filePaths.forEach(filePath => addFileToList(filePath));
        })
        .catch(error => console.error('Error:', error));
};
