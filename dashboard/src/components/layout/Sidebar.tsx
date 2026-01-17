'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { 
  LayoutGrid, 
  Activity, 
  Layers,
  Search,
  Package,
} from 'lucide-react';

const navigation = [
  { name: 'Pipelines', href: '/', icon: LayoutGrid },
  { name: 'Traces', href: '/traces', icon: Activity },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="flex h-screen w-56 flex-col border-r border-[var(--border-primary)] bg-[var(--bg-secondary)]">
      {/* Logo */}
      <div className="flex h-14 items-center px-4 border-b border-[var(--border-primary)]">
        <Link href="/" className="flex items-center gap-2">
          <div className="flex items-center justify-center w-7 h-7 bg-[var(--accent)] rounded-md">
            <span className="text-white font-bold text-sm">X</span>
          </div>
          <span className="text-[15px] font-semibold text-[var(--text-primary)]">X-Ray</span>
        </Link>
      </div>

      {/* Search */}
      <div className="p-3 border-b border-[var(--border-primary)]">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-tertiary)]" />
          <input
            type="text"
            placeholder="Search..."
            className="w-full h-8 pl-8 pr-3 text-sm bg-[var(--bg-tertiary)] border-0 rounded-md"
          />
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-3 space-y-0.5">
        {navigation.map((item) => {
          const isActive = pathname === item.href || 
            (item.href !== '/' && pathname.startsWith(item.href));
          
          return (
            <Link
              key={item.name}
              href={item.href}
              className={`
                flex items-center gap-2.5 px-2.5 py-2 rounded-md text-[13px] font-medium
                transition-colors duration-100
                ${isActive 
                  ? 'bg-[var(--accent)]/10 text-[var(--accent)]' 
                  : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]'
                }
              `}
            >
              <item.icon className="w-4 h-4" />
              {item.name}
            </Link>
          );
        })}
      </nav>

      {/* Footer */}
      <div className="p-3 border-t border-[var(--border-primary)]">
        <div className="flex items-center gap-2 text-xs text-[var(--text-tertiary)]">
          <span className="w-1.5 h-1.5 bg-[var(--success)] rounded-full" />
          API Connected
        </div>
      </div>
    </aside>
  );
}
