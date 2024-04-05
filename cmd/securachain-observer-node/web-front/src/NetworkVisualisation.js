import React, { useEffect, useRef, useState } from 'react';
import * as d3 from 'd3';

const NetworkVisualisation = ({ nodes, links, highlightNodeId, pubsubSignals, selectedTopic }) => {
    const svgRef = useRef(null);
    const [dimensions, setDimensions] = useState({
        width: window.innerWidth,
        height: window.innerHeight
    });
    const [selectedNode, setSelectedNode] = useState(null);
    const [clickPosition, setClickPosition] = useState({ x: 0, y: 0 });
    const linksDefaultColor = '#999';
    const linksHighlightedColor = 'red';
    const nodesDefaultColor = 'blue';
    const nodesHighlightedColor = 'red';
    const nodesConnectedColor = 'orange';
    const storageNodesColor = 'lightgreen';
    const fullNodesColor = 'mediumorchid';
    const clientNodesColor = 'lightcoral';
    const miningNodesColor = 'gold';
    const bootstrapNodesColor = 'midnightblue';
    const observerNodesColor = 'lightpink';

    useEffect(() => {
        // Fonction pour mettre à jour les dimensions
        const updateDimensions = () => {
            setDimensions({
                width: window.innerWidth,
                height: window.innerHeight
            });
        };

        // Écouter le redimensionnement de la fenêtre
        window.addEventListener('resize', updateDimensions);

        // Nettoyage de l'effet
        return () => window.removeEventListener('resize', updateDimensions);
    }, []);

    useEffect(() => {
        const { width, height } = dimensions;
        const svg = d3.select(svgRef.current);
        svg.attr('width', width).attr('height', height);

        let g = svg.select('g');
        if (g.empty()) {
            g = svg.append('g');
        }

        let linkG = g.select('g.links');
        if (linkG.empty()) {
            linkG = g.append('g').classed('links', true);
        }

        let nodeG = g.select('g.nodes');
        if (nodeG.empty()) {
            nodeG = g.append('g').classed('nodes', true);
        }

        const zoom = d3.zoom()
            .scaleExtent([1 / 2, 4])
            .on('zoom', (event) => {
                g.attr('transform', event.transform);
            });
        svg.call(zoom);

        // Configuration initiale de la simulation, sans spécifier de données
        const simulation = d3.forceSimulation()
            .force('link', d3.forceLink().id(d => d.id).distance(50))
            .force('charge', d3.forceManyBody().strength(-300))
            .force('center', d3.forceCenter(width / 2, height / 2))
            .force('collide', d3.forceCollide().radius(100));

        // Fonction de mise à jour des données de la simulation
        const updateSimulationData = () => {
            const highlightNode = nodes.find(node => node.id === highlightNodeId);
            let connectedNodeIds = new Set();
            if (highlightNode) {
                links.forEach(link => {
                    if (link.source.id === highlightNodeId || link.target.id === highlightNodeId) {
                        connectedNodeIds.add(link.source.id);
                        connectedNodeIds.add(link.target.id);
                    } else if (link.source === highlightNodeId || link.target === highlightNodeId) {
                        connectedNodeIds.add(link.source);
                        connectedNodeIds.add(link.target);
                    }
                });
            }
            // Mise à jour des liens
            const link = linkG.selectAll('line')
                .data(links, d => `${d.source.id}-${d.target.id}`);
            link.enter().append('line').merge(link)
                .style('stroke', link => {
                    // Si le lien est mis en évidence, changez sa couleur en rouge et stockez cette information dans les données
                    if ((link.source.id === highlightNodeId || link.target.id === highlightNodeId) || (link.source === highlightNodeId || link.target === highlightNodeId)) {
                        link.highlighted = true;
                        return linksHighlightedColor;
                    } else {
                        link.highlighted = false;
                        return linksDefaultColor;
                    }
                })
                .style('stroke-opacity', 0.6);
            link.exit().remove();

            // Mise à jour des nœuds
            const node = nodeG.selectAll('circle')
                .data(nodes, d => d.id);
            node.enter().append('circle').merge(node)
                .attr('r', 5)
                .style('fill', node => {
                    if (node.id === highlightNodeId) { 
                        return nodesHighlightedColor; // Couleur pour le nœud mis en évidence
                    } else if (connectedNodeIds.has(node.id)) {
                        return nodesConnectedColor; // Couleur pour les nœuds connectés
                    } else if (node.nodeType === 'StorageNode') {
                        return storageNodesColor; // Couleur pour les nœuds de stockage
                    } else if (node.nodeType === 'FullNode') {
                        return fullNodesColor; // Couleur pour les nœuds complets
                    } else if (node.nodeType === 'ClientNode') {
                        return clientNodesColor; // Couleur pour les nœuds clients
                    } else if (node.nodeType === 'MiningNode') {
                        return miningNodesColor; // Couleur pour les nœuds de minage
                    } else if (node.nodeType === 'BootstrapNode') {
                        return bootstrapNodesColor; // Couleur pour les nœuds de démarrage
                    } else if (node.nodeType === 'ObserverNode') {
                        return observerNodesColor; // Couleur pour les nœuds observateurs
                    } else { 
                        return nodesDefaultColor; // Couleur par défaut pour les nœuds
                    }
                })
                .on('click', (event, d) => {
                    setSelectedNode(d);
                    setClickPosition({ x: event.pageX, y: event.pageY });
                });
            node.exit().remove();

            // Appliquer les données mises à jour à la simulation
            simulation.nodes(nodes).on('tick', ticked);
            simulation.force('link').links(links);
            simulation.alpha(1).restart();
        };

        // Fonction pour positionner les éléments à chaque tick de la simulation
        const ticked = () => {
            nodeG.selectAll('circle')
                .attr('cx', d => d.x)
                .attr('cy', d => d.y);
            
            linkG.selectAll('line')
                .attr('x1', d => d.source.x)
                .attr('y1', d => d.source.y)
                .attr('x2', d => d.target.x)
                .attr('y2', d => d.target.y);

        };

        updateSimulationData();

        // Zoom and center on the highlighted node
        if (highlightNodeId) {
            const highlightNode = nodes.find(node => node.id === highlightNodeId);
            if (highlightNode) {
                const scale = 1; // Example zoom scale
                const translate = [
                    window.innerWidth / 2 - scale * highlightNode.x,
                    window.innerHeight / 2 - scale * highlightNode.y
                ];
                svg.transition()
                    .duration(750)
                    .call(
                        zoom.transform,
                        d3.zoomIdentity.translate(translate[0], translate[1]).scale(scale)
                    );
            }
        }

        // Créez une fonction pour animer les noeuds
        function animateNodes() {
            nodes.forEach((node, i) => {
                // Appliquez une petite perturbation basée sur un simple calcul trigonométrique pour simuler un mouvement organique
                node.fx = node.x + 0.1 * Math.sin(Date.now() / 1000 + i);
                node.fy = node.y + 0.1 * Math.cos(Date.now() / 1000 + i);
            });

            // Appliquez manuellement la logique de tick pour mettre à jour les positions sans réinitialiser la simulation
            simulation.nodes(nodes); // Assurez-vous de remettre à jour les nœuds dans la simulation
            simulation.alpha(0.3).restart(); // Redémarrez la simulation avec une faible alpha pour éviter des mouvements brusques
        }

        // Lancez l'animation des noeuds
        const intervalId = setInterval(animateNodes, 40); // Mettez à jour environ toutes les 40ms pour un mouvement fluide

        return () => {
            clearInterval(intervalId);
            if (simulation) {
                simulation.stop();
            }
        };
    }, [nodes, links, dimensions, highlightNodeId]); // Exécuter à chaque changement des props `nodes` et `links` et des dimensions

    useEffect(() => {
        const svg = d3.select(svgRef.current);
    
        // Définissez les données de la légende
        const legendData = [
            { color: nodesDefaultColor, text: "Default Node" },
            { color: nodesHighlightedColor, text: "Highlighted Node" },
            { color: nodesConnectedColor, text: "Connected Node" },
            { color: storageNodesColor, text: "Storage Node" },
            { color: fullNodesColor, text: "Full Node" },
            { color: clientNodesColor, text: "Client Node" },
            { color: miningNodesColor, text: "Mining Node" },
            { color: bootstrapNodesColor, text: "Bootstrap Node" },
            { color: observerNodesColor, text: "Observer Node" },
        ];
    
        // Ajoutez un groupe SVG pour la légende
        let legend = svg.select(".legend");
        if (legend.empty()) {
            legend = svg.append("g").classed("legend", true)
                         .attr("transform", "translate(10, " + (dimensions.height / 2) + ")");
        }
        
        // Ajoutez un fond blanc avec une bordure grise pour la légende
        const legendPadding = 5;
        const legendBackground = legend.select(".legend-background");
        if (legendBackground.empty()) {
            legend.insert("rect", ":first-child")
                .classed("legend-background", true)
                .attr("x", -legendPadding)
                .attr("y", -legendPadding)
                .attr("width", 160) // Assurez-vous que cette largeur est suffisante pour tout le texte
                .attr("height", legendData.length * 20 + 2 * legendPadding)
                .style("fill", "white")
                .style("stroke", "grey")
                .style("stroke-width", "1px");
        }

        // Ajoutez des rectangles et du texte pour chaque type de nœud
        const legendItem = legend.selectAll(".legend-item")
            .data(legendData)
            .enter()
            .append("g")
            .classed("legend-item", true)
            .attr("transform", (d, i) => `translate(0, ${i * 20})`);
    
        legendItem.append("rect")
            .attr("width", 18)
            .attr("height", 18)
            .style("fill", d => d.color);
    
        legendItem.append("text")
            .attr("x", 24)
            .attr("y", 9)
            .attr("dy", ".35em")
            .text(d => d.text);
    
        // Mettez à jour la fonction de mise à jour de la simulation et d'autres logiques si nécessaire...
    }, [dimensions.height]); // S'assurer que la légende se repositionne correctement si la hauteur de la fenêtre change
    
    useEffect(() => {
        const svg = d3.select(svgRef.current);
        const visited = new Set(); // Ensemble pour suivre les nœuds déjà visités
        const linkG = svg.select('g.links');
    
        // Fonction pour propager le signal à partir d'un nœud source
        const propagateSignal = (sourceNodeId, level = 0) => {
            // Trouvez le nœud source par ID
            const sourceNode = nodes.find(n => n.id === sourceNodeId);
            
            if (sourceNode && !visited.has(sourceNode.id)) {
                visited.add(sourceNode.id); // Marquez le nœud comme visité
    
                // Trouvez tous les nœuds directement connectés au nœud source
                const connections = links.filter(link => 
                    link.source.id === sourceNodeId || link.target.id === sourceNodeId
                );
    
                connections.forEach(connection => {
                    const targetNodeId = connection.source.id === sourceNodeId ? connection.target.id : connection.source.id;
                    const targetNode = nodes.find(n => n.id === targetNodeId);
    
                    // Assurez-vous que le nœud cible existe et n'a pas été visité
                    if (targetNode && !visited.has(targetNode.id)) {
                        // Créez un élément temporaire pour l'animation
                        const path = linkG.append('path')
                            .attr('d', `M${sourceNode.x},${sourceNode.y} L${targetNode.x},${targetNode.y}`)
                            .attr('stroke', 'black')
                            .attr('stroke-width', 1)
                            .attr('fill', 'none');

                        const totalLength = path.node().getTotalLength();
                        path.attr('stroke-dasharray', totalLength + ' ' + totalLength)
                            .attr('stroke-dashoffset', totalLength)
                            .transition()
                            .duration(2500)
                            .ease(d3.easeLinear)
                            .attr('stroke-dashoffset', 0)
                            .transition()
                            .duration(1000)
                            .ease(d3.easeLinear)
                            .attr('stroke-dashoffset', -totalLength)
                            .on('end', function() {
                                path.remove(); // Supprimez l'élément après l'animation
                                propagateSignal(targetNode.id, level + 1); // Propagez récursivement à partir du nœud cible
                            });
                    }
                });
            }
        };
        
        pubsubSignals.forEach(signal => {
            propagateSignal(signal.from); // Commencez la propagation à partir du nœud d'origine de chaque signal
        });
    
    }, [pubsubSignals, links, nodes, selectedTopic]);      
    
    return (
        <>
            <svg ref={svgRef}></svg>
            {selectedNode && (
                <div
                    style={{
                        position: 'absolute',
                        left: `${clickPosition.x}px`,
                        top: `${clickPosition.y}px`,
                        background: 'white',
                        border: '1px solid black',
                        padding: '10px'
                    }}
                >
                    <p>ID: {selectedNode.id}</p>
                    <p>Node type: {selectedNode.nodeType}</p>
                    <p>Topics: {selectedNode.topicsList.join(', ')}</p>
                    <button onClick={() => setSelectedNode(null)}>Close</button>
                </div>
            )}
        </>
    );
};

export default NetworkVisualisation;