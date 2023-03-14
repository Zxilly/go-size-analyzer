import './style.css'

import * as d3 from "d3";
import "d3-hierarchy";

async function loadCsv() {
    return await d3.csv('/csv');
}

const getID = () => Math.random().toString(36).substring(7);

async function main() {
    const csv = await loadCsv();

    const csvData = d3.stratify()
        .id(d => d.id)
        .parentId(d => {
            return d.parent_id
        })
        (csv)
        .each(node => {
            node.value = +node.data.size;
        })
        .sort((a, b) => b.size - a.size);
    drawMap(csvData);

    addEventListener("resize", () => {
        document.querySelector("div#chart").children[0].remove();
        drawMap(csvData);
    });
}

async function drawMap(csvData) {
    let width = window.innerWidth;
    let height = window.innerHeight - 60;

    const treemap = (data) => {
        return d3.treemap()
            .tile(tile)
            (data)
    }

    const algo = d3.treemapBinary;
    function tile(node, x0, y0, x1, y1) {
        algo(node, 0, 0, width, height);
        for (const child of node.children) {
            child.x0 = x0 + child.x0 / width * (x1 - x0);
            child.x1 = x0 + child.x1 / width * (x1 - x0);
            child.y0 = y0 + child.y0 / height * (y1 - y0);
            child.y1 = y0 + child.y1 / height * (y1 - y0);
        }
    }

    const x = d3.scaleLinear().rangeRound([0, width]);
    const y = d3.scaleLinear().rangeRound([0, height]);

    const svg = d3.select("div#chart")
        .append("svg")
        .attr("preserveAspectRatio", "xMinYMin meet")
        .attr("viewBox", [0.5, -60.5, width, height + 60])
        .attr("style", "max-height: 100%; width: intrinsic;")
        .style("font", "15px sans-serif")


    let group = svg.append("g")
        .call(render, treemap(csvData));

    function format_size(size) {
        let unit = "B";
        let fsize = parseFloat(size);
        if (fsize > 1024) {
            fsize /= 1024;
            unit = "KB";
        }
        if (fsize > 1024) {
            fsize /= 1024;
            unit = "MB";
        }
        if (fsize > 1024) {
            fsize /= 1024;
            unit = "GB";
        }
        return `${fsize.toFixed(2)} ${unit}`;
    }

    const colorMap = {
        0: "#E8EAF6",
        1: "#C5CAE9",
        2: "#9FA8DA",
        3: "#7986CB",
        4: "#5C6BC0",
        5: "#3F51B5",
        6: "#3949AB",
        7: "#303F9F",
        8: "#283593",
        9: "#1A237E",
    }

    function display(node) {
        if (node.data.id === "ROOT") {
            return "root";
        }
        return node.data.display_name;
    }

    function render(group, root) {
        const node = group
            .selectAll("g")
            .data(root.children.concat(root))
            .join("g");

        node.filter(d => d === root ? d.parent : d.children)
            .attr("cursor", "pointer")
            .on("click", (event, d) => d === root ? zoomout(root) : zoomin(d));

        node.append("title")
            .text(d => {
                return `${display(d)}\n${format_size(d.data.size)}`
            })

        node.append("rect")
            .attr("id", d => (d.leafUid = getID()))
            .attr("fill", d => d === root ? "#fff" : d.children ? "#90CAF9" : "#E3F2FD")
            .attr("stroke", "#fff");

        node.append("clipPath")
            .attr("id", d => (d.clipUid = getID()))
            .append("use")
            .attr("xlink:href", d => d.leafUid.href);

        node.append("text")
            .attr("clip-path", d => d.clipUid)
            .attr("font-weight", d => d === root ? "bold" : null)
            .selectAll("tspan")
            .data(d => {
                return [display(d), format_size(d.data.size)];
            })
            .join("tspan")
            .attr("x", 3)
            .attr("y", (d, i, nodes) => `${(i === nodes.length - 1) * 0.3 + 1.1 + i * 0.9}em`)
            .attr("fill-opacity", (d, i, nodes) => i === nodes.length - 1 ? 0.7 : null)
            .attr("font-weight", (d, i, nodes) => i === nodes.length - 1 ? "normal" : null)
            .text(d => {
                return d;
            });

        group.call(position, root);
    }

    function position(group, root) {
        group.selectAll("g")
            .attr("transform", d => d === root ? `translate(0,-60)` : `translate(${x(d.x0)},${y(d.y0)})`)
            .select("rect")
            .attr("width", d => d === root ? width : x(d.x1) - x(d.x0))
            .attr("height", d => d === root ? 60 : y(d.y1) - y(d.y0));
    }

    function zoomin(d) {
        const group0 = group.attr("pointer-events", "none");
        const group1 = group = svg.append("g").call(render, d);

        x.domain([d.x0, d.x1]);
        y.domain([d.y0, d.y1]);

        svg.transition()
            .duration(750)
            .call(t => group0.transition(t).remove()
                .call(position, d.parent))
            .call(t => group1.transition(t)
                .attrTween("opacity", () => d3.interpolate(0, 1))
                .call(position, d));
    }

    function zoomout(d) {
        const group0 = group.attr("pointer-events", "none");
        const group1 = group = svg.insert("g", "*").call(render, d.parent);

        x.domain([d.parent.x0, d.parent.x1]);
        y.domain([d.parent.y0, d.parent.y1]);

        svg.transition()
            .duration(750)
            .call(t => group0.transition(t).remove()
                .attrTween("opacity", () => d3.interpolate(1, 0))
                .call(position, d))
            .call(t => group1.transition(t)
                .call(position, d.parent));
    }
}


addEventListener('DOMContentLoaded', main)
