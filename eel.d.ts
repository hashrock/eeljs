declare const eel: {
  args: string[];
  exit(code: number): never;
  readTextFile(path: string): string;
  writeTextFile(path: string, text: string): void;
  env(name: string): string | null;
  fetch(
    url: string,
    init?: { method?: string; headers?: Record<string, string>; body?: string },
  ): { status: number; headers: Record<string, string>; text: string };
  run(cmd: string, args: string[]): { code: number; stdout: string; stderr: string };
};
