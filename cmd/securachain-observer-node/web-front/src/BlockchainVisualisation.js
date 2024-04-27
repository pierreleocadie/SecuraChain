import React, { useEffect, useRef, useState } from 'react';
import * as d3 from 'd3';
import ELK from 'elkjs/lib/elk.bundled.js';

const elk = new ELK();

const BlockchainVisualisation = ({ blocks, chains }) => {
    const svgRef = useRef(null);
    const modalRef = useRef(null);
    const [selectedBlock, setSelectedBlock] = useState(null);
    const [dimensions, setDimensions] = useState({
        width: window.innerWidth,
        height: window.innerHeight
    });

    useEffect(() => {
        const updateDimensions = () => {
            setDimensions({
                width: window.innerWidth,
                height: window.innerHeight
            });
        };

        window.addEventListener('resize', updateDimensions);
        return () => window.removeEventListener('resize', updateDimensions);
    }, []);

    useEffect(() => {
        const { width, height } = dimensions;

        const nodes = Object.values(blocks).map(block => ({
            id: block.hash,
            width: 100,
            height: 60,
            label: `Block ${block.height}`
        }));

        const edges = chains
            .map(chain => ({
                id: `edge-${chain.currentHash}`,
                sources: [chain.currentHash],
                targets: blocks[chain.previousHash] ? [chain.previousHash] : []
            }))
            .filter(edge => edge.targets.length > 0);

        const graph = {
            id: "root",
            children: nodes,
            edges: edges,
            layoutOptions: {
                'elk.algorithm': 'layered',
                'elk.direction': 'DOWN'
            }
        };

        elk.layout(graph).then(layout => {
            const svg = d3.select(svgRef.current)
                .attr('width', width)
                .attr('height', height);

            const zoom = d3.zoom()
                .scaleExtent([0.5, 2])
                .on("zoom", (event) => svg.selectAll('g').attr("transform", event.transform));

            svg.call(zoom);
            svg.selectAll("*").remove();

            const g = svg.append("g");

            g.selectAll("line")
                .data(layout.edges)
                .enter()
                .append("line")
                .attr("x1", d => layout.children.find(n => n.id === d.sources[0]).x + 50)
                .attr("y1", d => layout.children.find(n => n.id === d.sources[0]).y + 30)
                .attr("x2", d => layout.children.find(n => n.id === d.targets[0]).x + 50)
                .attr("y2", d => layout.children.find(n => n.id === d.targets[0]).y + 30)
                .attr("stroke", "black");

            g.selectAll("rect")
                .data(layout.children)
                .enter()
                .append("rect")
                .attr("x", d => d.x)
                .attr("y", d => d.y)
                .attr("width", d => d.width)
                .attr("height", d => d.height)
                .attr("fill", "lightblue")
                .attr("stroke", "black")
                .on("click", (event, d) => {
                    setSelectedBlock(blocks[d.id]);
                    event.stopPropagation();
                });

            g.selectAll("text")
                .data(layout.children)
                .enter()
                .append("text")
                .attr("x", d => d.x + 50)
                .attr("y", d => d.y + 35)
                .attr("text-anchor", "middle")
                .text(d => d.label);
        });
    }, [blocks, chains, dimensions]);

    const closeModal = () => {
        setSelectedBlock(null);
    };

    useEffect(() => {
        const handleOutsideClick = (event) => {
            if (modalRef.current && !modalRef.current.contains(event.target)) {
                closeModal();
            }
        };

        if (selectedBlock) {
            document.addEventListener("click", handleOutsideClick);
        }

        return () => {
            document.removeEventListener("click", handleOutsideClick);
        };
    }, [selectedBlock]);

    return (
        <>
            <svg ref={svgRef}></svg>
            {selectedBlock && (
                <div ref={modalRef} style={{
                    position: 'fixed',
                    top: '50%',
                    left: '50%',
                    transform: 'translate(-50%, -50%)',
                    background: 'white',
                    padding: '20px',
                    border: '1px solid black',
                    boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
                    zIndex: 1000
                }}>
                    <h2>Block Details</h2>
                    <p><b>Hash:</b> {selectedBlock.hash}</p>
                    <p><b>Version:</b> {selectedBlock.version}</p>
                    <p><b>Previous:</b> {selectedBlock.prev_block}</p>
                    <p><b>Merkle root:</b> {selectedBlock.merkle_root}</p>
                    <p><b>Target bits:</b> {selectedBlock.target_bits}</p>
                    <p><b>Timestamp:</b> {selectedBlock.timestamp}</p>
                    <p><b>Height:</b> {selectedBlock.height}</p>
                    <p><b>Nonce:</b> {selectedBlock.nonce}</p>
                    <p><b>Miner Address:</b> {selectedBlock.miner_addr}</p>
                    <p><b>Signature:</b> {selectedBlock.signature}</p>
                    <p><b>Number of transactions:</b> {selectedBlock.transactions ? selectedBlock.transactions.length : 0}</p>
                    <h3>Transactions</h3>
                    {selectedBlock.transactions && (
                        <ul>
                            {selectedBlock.transactions.map((tx, index) => (
                                <li key={index}>{tx.transactionID}</li>
                            ))}
                        </ul>
                    )}
                    <button onClick={closeModal}>Close</button>
                </div>
            )}
            {selectedBlock && (
                <div style={{
                    position: 'fixed',
                    top: 0,
                    left: 0,
                    width: '100%',
                    height: '100%',
                    backgroundColor: 'rgba(0, 0, 0, 0.5)'
                }}></div>
            )}
        </>
    );
};

export default BlockchainVisualisation;