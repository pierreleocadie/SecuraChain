const express = require('express');
const multer = require('multer');
const fs = require('fs');
const path = require('path');
const app = express();

// Configuration de stockage Multer
const storage = multer.diskStorage({
    destination: function(req, file, cb) {
        cb(null, 'uploads/');
    },
    filename: function(req, file, cb) {
        cb(null, file.originalname); // Utiliser le nom original du fichier
    }
});

const upload = multer({ storage: storage });
const uploadDirectory = path.join(__dirname, 'uploads');

app.use(express.static('.'));

app.post('/upload', upload.single('file'), function(req, res) {
    res.json({ filePath: `/uploads/${req.file.originalname}`, fileName: req.file.originalname });
});

app.get('/files', function(req, res) {
    fs.readdir(uploadDirectory, function(err, files) {
        if (err) {
            res.sendStatus(500);
        } else {
            const fileInfos = files.map(file => {
                return { filePath: `/uploads/${file}`, fileName: file };
            });
            res.json(fileInfos);
        }
    });
});

app.delete('/delete/:fileName', function(req, res) {
    const fileName = req.params.fileName;
    fs.unlink(path.join(uploadDirectory, fileName), (err) => {
        if (err) {
            console.error(err);
            return res.sendStatus(500);
        }
        res.sendStatus(200);
    });
});

app.listen(3000, function() {
    console.log('App listening on port 3000!');
});
