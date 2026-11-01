export class ApiError extends Error {
  public readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

const STATUS_MESSAGES: Record<number, string> = {
  400: "Les informations saisies sont invalides.",
  401: "Identifiants ou session invalides.",
  403: "Action non autorisée.",
  404: "Élément introuvable.",
  409: "Cette action a déjà été effectuée.",
  413: "Le fichier est trop volumineux.",
  415: "Format de fichier non pris en charge.",
  429: "Trop de tentatives. Réessayez dans un instant.",
  500: "Une erreur est survenue. Réessayez plus tard.",
};

/** Message utilisateur (francais) pour toute erreur. */
export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 0) {
      return error.message;
    }
    return STATUS_MESSAGES[error.status] ?? "Une erreur est survenue.";
  }
  return "Une erreur est survenue.";
}
