import { useEffect, useRef, useState } from 'react';
import { motion, useAnimation } from 'motion/react';

interface CircularTextProps {
  text: string;
  spinDuration?: number;
  onHover?: 'slowDown' | 'speedUp' | 'pause' | 'goBonkers';
  direction?: 'clockwise' | 'counterclockwise';
  className?: string;
  size?: number;
  fontSize?: number;
  fontWeight?: number;
  textColor?: string;
  radius?: number;
  draggable?: boolean;
}

interface DragInfo {
  delta: { x: number };
  velocity: { x: number };
}

export default function CircularText({
  text,
  spinDuration = 20,
  onHover = 'speedUp',
  direction = 'clockwise',
  className = '',
  size = 200,
  fontSize = 24,
  fontWeight = 900,
  textColor = '#087c60',
  draggable = false,
}: CircularTextProps) {
  const letters = Array.from(text);
  const controls = useAnimation();
  const currentRotation = useRef(0);
  const [isDragging, setIsDragging] = useState(false);

  useEffect(() => {
    if (!isDragging) {
      controls.start({
        rotate: direction === 'clockwise'
          ? currentRotation.current + 360
          : currentRotation.current - 360,
        transition: {
          duration: spinDuration,
          ease: 'linear',
          repeat: Infinity,
        },
      });
    }
  }, [spinDuration, direction, controls, isDragging, currentRotation]);

  return (
    <div
      className={`relative flex items-center justify-center rounded-full cursor-grab active:cursor-grabbing ${className}`}
      style={{ width: size, height: size }}
    >
      <motion.div
        className="w-full h-full relative flex items-center justify-center"
        animate={controls}
        onUpdate={(latest: { rotate?: number }) => {
          if (typeof latest.rotate === 'number') {
            currentRotation.current = latest.rotate;
          }
        }}
        {...(draggable
          ? {
              drag: true,
              dragConstraints: { left: 0, right: 0, top: 0, bottom: 0 },
              onDragStart: () => {
                setIsDragging(true);
                controls.stop();
              },
              onDrag: (_event: unknown, info: DragInfo) => {
                currentRotation.current += info.delta.x * 0.5;
              },
              onDragEnd: (_event: unknown, info: DragInfo) => {
                setIsDragging(false);
                const velocity = info.velocity.x * 0.1;
                controls.start({
                  rotate: currentRotation.current + velocity * 360,
                  transition: {
                    duration: spinDuration,
                    ease: 'easeOut',
                  },
                });
              },
            }
          : {})}
        whileHover={
          onHover !== 'pause'
            ? {
                scale: onHover === 'goBonkers' ? 0.9 : 1.05,
                transition: { duration: 0.3, ease: 'easeOut' },
              }
            : {}
        }
      >
        {letters.map((letter, i) => {
          const angle = (i / letters.length) * 360;
          const radian = (angle * Math.PI) / 180;
          const radius = size * 0.35;
          const x = radius * Math.cos(radian);
          const y = radius * Math.sin(radian);

          return (
            <span
              key={i}
              className="absolute inline-block whitespace-nowrap select-none"
              style={{
                fontSize: `${fontSize}px`,
                fontWeight: fontWeight,
                color: textColor,
                transform: `translate(${x}px, ${y}px) rotate(${angle + 90}deg)`,
              }}
            >
              {letter}
            </span>
          );
        })}
      </motion.div>
    </div>
  );
}