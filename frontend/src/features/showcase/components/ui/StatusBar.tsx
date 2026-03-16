export function StatusBar() {
  return (
    <div className="flex items-center justify-between h-[62px] px-6">
      <span className="text-sm font-bold text-text-primary">9:41</span>
      <div className="flex items-center gap-1.5 text-xs text-text-primary">
        <span>▂▄▆█</span>
        <span>⏣</span>
        <span>🔋</span>
      </div>
    </div>
  );
}
