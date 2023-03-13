import './style.css'

import * as d3 from "d3";
import { stratify, treemapBinary } from "d3-hierarchy";

async function loadCsv() {
    return await d3.csv('http://127.0.0.1:8888/csv');
}

const getID = () => Math.random().toString(36).substring(7);

async function main() {
    const width = 960;
    const height = 500;

    const csv = await loadCsv();

    const csvData = stratify()
        .id(d => d.id)
        .parentId(d => {
            return d.parent_id
        })
        (csv)
        .sum(d => +d.size)
        .sort((a, b) => b.size - a.size);

    const treemap = (data) => {
        return d3.treemap()
            .tile(tile)
            (data)
    }

    function tile(node, x0, y0, x1, y1) {
        d3.treemapBinary(node, 0, 0, width, height);
        for (const child of node.children) {
            child.x0 = x0 + child.x0 / width * (x1 - x0);
            child.x1 = x0 + child.x1 / width * (x1 - x0);
            child.y0 = y0 + child.y0 / height * (y1 - y0);
            child.y1 = y0 + child.y1 / height * (y1 - y0);
        }
    }

    const x = d3.scaleLinear().rangeRound([0, width]);
    const y = d3.scaleLinear().rangeRound([0, height]);

    const svg = d3.select("body").append("svg")
        .attr("viewBox", [0.5, -30.5, width, height + 30])
        .style("font", "10px sans-serif");

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

    function render(group, root) {
        const node = group
            .selectAll("g")
            .data(root.children.concat(root))
            .join("g");

        node.filter(d => d === root ? d.parent : d.children)
            .attr("cursor", "pointer")
            .on("click", (event, d) => d === root ? zoomout(root) : zoomin(d));

        node.append("title")
            .text(d => d)

        node.append("rect")
            .attr("id", d => (d.leafUid = getID()))
            .attr("fill", d => d === root ? "#fff" : d.children ? "#ccc" : "#ddd")
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
                if (d.data.id == "ROOT") {
                    return ["root", format_size(d.data.size)];
                }

                return [`${d.data.display_name}`, format_size(d.data.size)];
            })
            .join("tspan")
            .attr("x", 3)
            .attr("y", (d, i, nodes) => `${(i === nodes.length - 1) * 0.3 + 1.1 + i * 0.9}em`)
            .attr("fill-opacity", (d, i, nodes) => i === nodes.length - 1 ? 0.7 : null)
            .attr("font-weight", (d, i, nodes) => i === nodes.length - 1 ? "normal" : null)
            .text(d => {
                console.log(d);
                return d;
            });

        group.call(position, root);
    }

    function position(group, root) {
        group.selectAll("g")
            .attr("transform", d => d === root ? `translate(0,-30)` : `translate(${x(d.x0)},${y(d.y0)})`)
            .select("rect")
            .attr("width", d => d === root ? width : x(d.x1) - x(d.x0))
            .attr("height", d => d === root ? 30 : y(d.y1) - y(d.y0));
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
