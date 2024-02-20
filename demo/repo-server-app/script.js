function uploadFile() {
    var formData = new FormData(document.getElementById('uploadForm'));
    fetch('/upload', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        if(data.filePath) {
            addFileToList(data.filePath, data.fileName);
        }
    })
    .catch(error => {
        console.error('Error:', error);
    });
}

function addFileToList(filePath, fileName) {
    var fileList = document.getElementById('fileList');
    var fileElement = document.createElement('div');
    
    var link = document.createElement('a');
    link.href = filePath;
    link.textContent = fileName;
    fileElement.appendChild(link);

    var deleteButton = document.createElement('button');
    deleteButton.textContent = 'Supprimer';
    deleteButton.onclick = function() {
        deleteFile(fileName);
    };
    fileElement.appendChild(deleteButton);

    fileList.appendChild(fileElement);
}

function deleteFile(fileName) {
    fetch(`/delete/${fileName}`, {
        method: 'DELETE',
    })
    .then(response => {
        if(response.ok) {
            window.location.reload(); // Recharger la page pour mettre à jour la liste des fichiers
        }
    })
    .catch(error => {
        console.error('Error:', error);
    });
}

window.onload = function() {
    fetch('/files')
        .then(response => response.json())
        .then(fileInfos => {
            fileInfos.forEach(fileInfo => addFileToList(fileInfo.filePath, fileInfo.fileName));
        })
        .catch(error => console.error('Error:', error));
};
