import React, { useState, useEffect } from 'react';
import BlockchainVisualisation from './BlockchainVisualisation';

function BlockchainData() {
    const [blockchainData, setBlockchainData] = useState({ blocks: {}, chains: [] });

    useEffect(() => {
        const ws = new WebSocket('ws://localhost:8080/ws');

        ws.onopen = () => {
            console.log('Connected to the WebSocket server');
        };

        ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            switch(message.type) {
                case 'initBlockchainData':
                    // Initialisation avec les données complètes
                    if (message.data.length > 0) {
                        processInitBlockchainData(message.data);
                    }
                    break;
                case 'newBlock':
                    // Mise à jour avec les différences
                    if (message.data) {
                        processNewBlock(message.data);
                    }
                    break;
                default:
                    console.log('Unknown message type');
            }
        };

        ws.onclose = () => {
            console.log('Disconnected from the WebSocket server');
        };

        // Cleanup function
        return () => {
            ws.close();
        };
    }, []);

    const processInitBlockchainData = (data) => {
        // Pour chaque block dans data on ajoute  `{ currentHash: ..., previousHash: ... }` à chains
        // On ajoute aussi chaque block à blocks avec la clé `currentHash` et la valeur le block lui-même
        for (const block of data) {
            setBlockchainData(currentData => {
                return {
                    blocks: { ...currentData.blocks, [block.hash]: block },
                    chains: [...currentData.chains, { currentHash: block.hash, previousHash: block.previous }]
                };
            });
        }
    }

    const processNewBlock = (data) => {
        // On ajoute le nouveau block à blocks avec la clé `currentHash` et la valeur le block lui-même
        // On ajoute aussi le block à chains avec la clé `currentHash` et la valeur le block précédent
        setBlockchainData(currentData => {
            return {
                blocks: { ...currentData.blocks, [data.hash]: data },
                chains: [...currentData.chains, { currentHash: data.hash, previousHash: data.previous }]
            };
        });
    }

    return (
        <div>
            <BlockchainVisualisation blocks={blockchainData.blocks} chains={blockchainData.chains} />
        </div>
    );
}

export default BlockchainData;
