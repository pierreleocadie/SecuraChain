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
    var fileRow = document.createElement('div');
    
    var fileLink = document.createElement('a');
    fileLink.href = filePath;
    fileLink.textContent = filePath.split('/').pop();
    fileRow.appendChild(fileLink);

    var deleteButton = document.createElement('button');
    deleteButton.textContent = 'Supprimer';
    deleteButton.onclick = function() { deleteFile(filePath); };
    fileRow.appendChild(deleteButton);

    fileList.appendChild(fileRow);
}

function deleteFile(filePath) {
    fetch(`/delete?filePath=${encodeURIComponent(filePath)}`, { method: 'DELETE' })
    .then(response => response.json())
    .then(data => {
        if(data.success) {
            // Recharger la liste des fichiers après la suppression
            document.getElementById('fileList').innerHTML = ''; // Effacer la liste actuelle
            fetch('/files') // Recharger la liste
                .then(response => response.json())
                .then(filePaths => {
                    filePaths.forEach(filePath => addFileToList(filePath));
                })
                .catch(error => console.error('Error:', error));
        }
    })
    .catch(error => console.error('Error:', error));
}

window.onload = function() {
    fetch('/files')
        .then(response => response.json())
        .then(filePaths => {
            filePaths.forEach(filePath => addFileToList(filePath));
        })
        .catch(error => console.error('Error:', error));
};
