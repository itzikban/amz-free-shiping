"use client";

import { useEffect, useRef } from "react";

interface Node {
  x: number;
  y: number;
  vx: number;
  vy: number;
  radius: number;
  pulse: number;
  pulseSpeed: number;
}

const NODE_COUNT = 65;
const CONNECTION_DIST = 160;
const MOUSE_RADIUS = 200;
const MOUSE_REPEL = 0.6;

export default function NeuralBackdrop() {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    let w = 0;
    let h = 0;
    let animId = 0;
    const mouse = { x: -9999, y: -9999 };
    let nodes: Node[] = [];

    function resize() {
      const dpr = window.devicePixelRatio || 1;
      w = window.innerWidth;
      h = window.innerHeight;
      canvas!.width = w * dpr;
      canvas!.height = h * dpr;
      canvas!.style.width = w + "px";
      canvas!.style.height = h + "px";
      ctx!.setTransform(dpr, 0, 0, dpr, 0, 0);
    }

    function createNodes() {
      nodes = [];
      for (let i = 0; i < NODE_COUNT; i++) {
        nodes.push({
          x: Math.random() * w,
          y: Math.random() * h,
          vx: (Math.random() - 0.5) * 0.4,
          vy: (Math.random() - 0.5) * 0.4,
          radius: Math.random() * 1.5 + 1,
          pulse: Math.random() * Math.PI * 2,
          pulseSpeed: 0.01 + Math.random() * 0.02,
        });
      }
    }

    function tick() {
      ctx!.clearRect(0, 0, w, h);

      for (const n of nodes) {
        n.pulse += n.pulseSpeed;

        const dx = n.x - mouse.x;
        const dy = n.y - mouse.y;
        const dist = Math.sqrt(dx * dx + dy * dy);
        if (dist < MOUSE_RADIUS && dist > 0) {
          const force = (1 - dist / MOUSE_RADIUS) * MOUSE_REPEL;
          n.vx += (dx / dist) * force;
          n.vy += (dy / dist) * force;
        }

        n.vx *= 0.99;
        n.vy *= 0.99;
        n.x += n.vx;
        n.y += n.vy;

        if (n.x < -20) n.x = w + 20;
        if (n.x > w + 20) n.x = -20;
        if (n.y < -20) n.y = h + 20;
        if (n.y > h + 20) n.y = -20;
      }

      // draw connections
      for (let i = 0; i < nodes.length; i++) {
        for (let j = i + 1; j < nodes.length; j++) {
          const a = nodes[i];
          const b = nodes[j];
          const ddx = a.x - b.x;
          const ddy = a.y - b.y;
          const dist = Math.sqrt(ddx * ddx + ddy * ddy);
          if (dist < CONNECTION_DIST) {
            const alpha = (1 - dist / CONNECTION_DIST) * 0.25;
            ctx!.beginPath();
            ctx!.moveTo(a.x, a.y);
            ctx!.lineTo(b.x, b.y);
            ctx!.strokeStyle = "rgba(71, 183, 255, " + alpha + ")";
            ctx!.lineWidth = 0.6;
            ctx!.stroke();
          }
        }
      }

      // draw mouse connections
      if (mouse.x > -999) {
        for (const n of nodes) {
          const ddx = n.x - mouse.x;
          const ddy = n.y - mouse.y;
          const dist = Math.sqrt(ddx * ddx + ddy * ddy);
          if (dist < MOUSE_RADIUS) {
            const alpha = (1 - dist / MOUSE_RADIUS) * 0.35;
            ctx!.beginPath();
            ctx!.moveTo(mouse.x, mouse.y);
            ctx!.lineTo(n.x, n.y);
            ctx!.strokeStyle = "rgba(155, 92, 255, " + alpha + ")";
            ctx!.lineWidth = 0.8;
            ctx!.stroke();
          }
        }
      }

      // draw nodes
      for (const n of nodes) {
        const glow = 0.4 + Math.sin(n.pulse) * 0.3;
        const r = n.radius + Math.sin(n.pulse) * 0.4;

        ctx!.beginPath();
        ctx!.arc(n.x, n.y, r + 3, 0, Math.PI * 2);
        ctx!.fillStyle = "rgba(33, 212, 253, " + (glow * 0.15) + ")";
        ctx!.fill();

        ctx!.beginPath();
        ctx!.arc(n.x, n.y, r, 0, Math.PI * 2);
        ctx!.fillStyle = "rgba(159, 232, 255, " + (glow + 0.3) + ")";
        ctx!.fill();
      }

      animId = requestAnimationFrame(tick);
    }

    function onMouseMove(e: MouseEvent) {
      mouse.x = e.clientX;
      mouse.y = e.clientY;
    }
    function onMouseLeave() {
      mouse.x = -9999;
      mouse.y = -9999;
    }
    function onTouchMove(e: TouchEvent) {
      if (e.touches.length > 0) {
        mouse.x = e.touches[0].clientX;
        mouse.y = e.touches[0].clientY;
      }
    }
    function onTouchEnd() {
      mouse.x = -9999;
      mouse.y = -9999;
    }

    resize();
    createNodes();
    tick();

    const onResize = () => { resize(); createNodes(); };
    window.addEventListener("resize", onResize);
    window.addEventListener("mousemove", onMouseMove);
    window.addEventListener("mouseleave", onMouseLeave);
    window.addEventListener("touchmove", onTouchMove);
    window.addEventListener("touchend", onTouchEnd);

    return () => {
      cancelAnimationFrame(animId);
      window.removeEventListener("resize", onResize);
      window.removeEventListener("mousemove", onMouseMove);
      window.removeEventListener("mouseleave", onMouseLeave);
      window.removeEventListener("touchmove", onTouchMove);
      window.removeEventListener("touchend", onTouchEnd);
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      className="neuralBackdrop"
      aria-hidden="true"
    />
  );
}
