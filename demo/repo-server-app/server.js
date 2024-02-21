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
        cb(null, file.originalname);
    }
});

const upload = multer({ storage: storage });
const uploadDirectory = path.join(__dirname, 'uploads');

// Assurer que le dossier d'upload existe
if (!fs.existsSync(uploadDirectory)){
    fs.mkdirSync(uploadDirectory);
}

app.use(express.static('.'));

app.post('/upload', upload.single('file'), function(req, res) {
    res.json({ filePath: `/uploads/${req.file.originalname}` });
});

app.get('/files', function(req, res) {
    fs.readdir(uploadDirectory, function(err, files) {
        if (err) {
            res.sendStatus(500);
        } else {
            const filePaths = files.map(file => `/uploads/${file}`);
            res.json(filePaths);
        }
    });
});

app.delete('/delete', function(req, res) {
    const filePath = path.join(__dirname, req.query.filePath);
    fs.unlink(filePath, function(err) {
        if (err) {
            console.error(err);
            res.status(500).json({success: false});
        } else {
            res.json({success: true});
        }
    });
});

app.listen(3000, function() {
    console.log('App listening on port 3000!');
});
