"use client";

export default function NeuralBackdrop() {
  const nodes = [
    [8, 18], [20, 30], [35, 16], [52, 28], [67, 20], [82, 32], [92, 18],
    [14, 62], [30, 52], [48, 64], [64, 56], [78, 68], [90, 58],
  ];

  const lines = [
    [0, 1], [1, 2], [2, 3], [3, 4], [4, 5], [5, 6],
    [7, 8], [8, 9], [9, 10], [10, 11], [11, 12],
    [1, 8], [3, 9], [4, 10], [5, 11], [6, 12],
  ];

  return (
    <div className="neuralBackdrop" aria-hidden="true">
      <svg viewBox="0 0 100 80" preserveAspectRatio="none">
        {lines.map(([a, b], idx) => (
          <line
            key={idx}
            x1={nodes[a][0]}
            y1={nodes[a][1]}
            x2={nodes[b][0]}
            y2={nodes[b][1]}
            className="neuralLine"
          />
        ))}
        {nodes.map(([x, y], idx) => (
          <circle key={idx} cx={x} cy={y} r="0.8" className="neuralNode" />
        ))}
      </svg>
      <div className="glowOrb orbA" />
      <div className="glowOrb orbB" />
      <div className="glowOrb orbC" />
    </div>
  );
}
