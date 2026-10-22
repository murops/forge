// same rules as the API, checked here only to give feedback before submitting
const NAME_RE = /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/;

export function validateName(name: string): string | undefined {
  if (name.length < 3 || name.length > 100) {
    return "Must be between 3 and 100 characters.";
  }
  if (!NAME_RE.test(name)) {
    return "Use lowercase letters, digits and hyphens, not starting or ending with a hyphen.";
  }
}

export function validateDescription(description: string): string | undefined {
  if (description.length > 350) {
    return "Must be at most 350 characters.";
  }
}
