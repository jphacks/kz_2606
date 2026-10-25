import { Footprints } from 'lucide-react';
import CircularText from './CircularText';

export default function AuthMapHero() {
  return (
    <div className="mb-6 h-48 rounded-2xl overflow-hidden shadow-md border border-[#D7E3D2] bg-[#F0EBDD] relative flex items-center justify-center">
      <svg
        aria-hidden="true"
        className="absolute inset-0 w-full h-full"
        viewBox="0 0 400 192"
        preserveAspectRatio="xMidYMid slice"
      >
        <rect width="400" height="192" fill="#F1EBDD" />
        <path d="M0 22H400M0 116H400M55 0V192M330 0V192" stroke="#E3DCCB" strokeWidth="1" />
        <path d="M22 0L90 192M260 0L190 192M400 72L270 192" stroke="#E3DCCB" strokeWidth="1" />
        <path d="M16 18h32v24H16zM67 11h27v31H67zM104 14h39v25h-39zM300 12h24v30h-24zM348 18h36v24h-36zM15 132h30v35H15zM75 136h43v28H75zM291 132h35v34h-35zM345 127h41v39h-41z" fill="#DCE8D2" />
        <path d="M0 72C76 68 107 90 166 76s104-24 234-8M140 0c-4 48 7 77 30 110s26 52 24 82M0 174c71-26 111-18 166-5s126 8 234-18" fill="none" stroke="#FFFDF8" strokeWidth="12" />
        <path d="M0 72C76 68 107 90 166 76s104-24 234-8M140 0c-4 48 7 77 30 110s26 52 24 82M0 174c71-26 111-18 166-5s126 8 234-18" fill="none" stroke="#DDD5C5" strokeWidth="1" />
        <path d="M35 119c52-34 80-21 121-37s72-13 102 4 55 17 91-2" fill="none" stroke="#16866A" strokeWidth="4" strokeLinecap="round" strokeDasharray="2 8" />
        <circle cx="35" cy="119" r="6" fill="#16866A" stroke="#FFFDF8" strokeWidth="3" />
        <circle cx="349" cy="84" r="6" fill="#E28A55" stroke="#FFFDF8" strokeWidth="3" />
      </svg>

      <div className="relative flex items-center justify-center w-[152px] h-[152px] rounded-full bg-white/75 shadow-sm ring-1 ring-white/90 backdrop-blur-[2px]">
        <div className="absolute inset-0 flex items-center justify-center">
          <CircularText
            text="まちえ • まちを歩こう • "
            onHover="speedUp"
            direction="clockwise"
            spinDuration={20}
            size={152}
            fontSize={11}
            textColor="#0F5A40"
            draggable
            fontWeight={900}
          />
        </div>

        <div className="absolute w-11 h-11 rounded-2xl bg-[#0F5A40] flex items-center justify-center text-white shadow-md z-10 pointer-events-none">
          <Footprints className="w-5 h-5 -rotate-45" />
        </div>
      </div>
    </div>
  );
}
