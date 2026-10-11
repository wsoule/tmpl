export function createGreeting(name = 'World'): string {
  const normalizedName = name.trim();
  const displayName = normalizedName.length > 0 ? normalizedName : 'World';
  return `Hello, ${displayName}!`;
}
