function uploadFile() {
    var formData = new FormData(document.getElementById('uploadForm'));
    fetch('/upload', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        if(data.filePath) {
            addFileToList(data.filePath);
        }
    })
    .catch(error => {
        console.error('Error:', error);
    });
}

function addFileToList(filePath) {
    var fileList = document.getElementById('fileList');
    var fileElement = document.createElement('a');
    fileElement.href = filePath;
    fileElement.textContent = filePath.split('/').pop();
    fileList.appendChild(fileElement);
    fileList.appendChild(document.createElement('br'));
}

window.onload = function() {
    fetch('/files')
        .then(response => response.json())
        .then(filePaths => {
            filePaths.forEach(filePath => addFileToList(filePath));
        })
        .catch(error => console.error('Error:', error));
};
