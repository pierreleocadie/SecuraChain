import React from 'react';
import { BrowserRouter, Routes, Route, Link } from 'react-router-dom';
import BlockchainData from './BlockchainData';
import NetworkData from './NetworkData';

const App = () => {
    return (
        <BrowserRouter>
            <div>
                <nav>
                    <Link to="/">Network Visualisation</Link>&nbsp;&nbsp;&nbsp;&nbsp;
                    <Link to="/blockchain">Blockchain Visualisation</Link>
                </nav>
                <Routes>
                    <Route path="/" element={<NetworkData />} />
                    <Route path="/blockchain" element={<BlockchainData />} />
                </Routes>
            </div>
        </BrowserRouter>
    );
    // <div className="App">
    //     <NetworkData/>
    // </div>
};

export default App;
