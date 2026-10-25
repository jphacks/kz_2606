import { useEffect, useRef, useState } from 'react';
import type { PointerEvent as ReactPointerEvent } from 'react';
import { animate, motion, useMotionValue } from 'motion/react';

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
  radius = size * 0.35,
  draggable = false,
}: CircularTextProps) {
  const letters = Array.from(text);
  const rotation = useMotionValue(0);
  const pointerId = useRef<number | null>(null);
  const lastPointerX = useRef(0);
  const [isDragging, setIsDragging] = useState(false);
  const [isHovered, setIsHovered] = useState(false);

  useEffect(() => {
    if (isDragging || (isHovered && onHover === 'pause')) {
      return;
    }

    const speedMultiplier = isHovered
      ? onHover === 'speedUp' ? 2.5 : onHover === 'slowDown' ? 0.4 : onHover === 'goBonkers' ? 5 : 1
      : 1;
    const targetRotation = rotation.get() + (direction === 'clockwise' ? 360 : -360);
    const animation = animate(rotation, targetRotation, {
      duration: spinDuration / speedMultiplier,
      ease: 'linear',
      repeat: Infinity,
    });

    return () => animation.stop();
  }, [direction, isDragging, isHovered, onHover, rotation, spinDuration]);

  const handlePointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (!draggable || (event.pointerType === 'mouse' && event.button !== 0)) {
      return;
    }

    event.currentTarget.setPointerCapture(event.pointerId);
    pointerId.current = event.pointerId;
    lastPointerX.current = event.clientX;
    setIsDragging(true);
  };

  const handlePointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (pointerId.current !== event.pointerId) {
      return;
    }

    rotation.set(rotation.get() + (event.clientX - lastPointerX.current) * 0.5);
    lastPointerX.current = event.clientX;
  };

  const handlePointerUp = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (pointerId.current !== event.pointerId) {
      return;
    }

    pointerId.current = null;
    setIsDragging(false);
  };

  return (
    <div
      className={`relative flex items-center justify-center rounded-full cursor-grab active:cursor-grabbing ${className}`}
      style={{ width: size, height: size, touchAction: draggable ? 'none' : undefined }}
    >
      <motion.div
        className="w-full h-full relative flex items-center justify-center"
        style={{ rotate: rotation }}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerUp}
        onHoverStart={() => setIsHovered(true)}
        onHoverEnd={() => setIsHovered(false)}
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