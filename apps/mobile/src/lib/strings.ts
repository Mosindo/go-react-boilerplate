/** User-facing copy in one place so the app stays i18n-ready. */
export const strings = {
  appName: "Alba",
  tagline: "Meet someone real. Always free.",
  freePromise: "Alba is 100% free: no subscriptions, no paywalls, no paid boosts. Ever.",
  auth: {
    welcomeTitle: "Find your person",
    welcomeBody: "Thoughtful matches, real conversations, and never a paywall.",
    signIn: "Sign in",
    createAccount: "Create account",
    loginTitle: "Welcome back",
    loginBody: "Sign in to pick up where you left off.",
    registerTitle: "Create your account",
    registerBody: "It takes a minute, and it is free.",
    forgotTitle: "Forgot your password?",
    forgotBody: "Enter your email and we will send you a reset token.",
    forgotSent:
      "If an account exists for that email, a reset token is on its way. Check your inbox, then paste the token below.",
    resetTitle: "Choose a new password",
    resetBody: "Paste the token from your email and pick a new password.",
    resetDone: "Password updated. Please sign in with your new password.",
    birthDateHint: "We only show your age, never your birth date. You must be 18 or older.",
    passwordHint: "At least 8 characters."
  },
  onboarding: {
    steps: ["About you", "Who you'd like to meet", "Your story", "Location", "Photos"],
    locationWhy:
      "Alba uses your approximate location to show people nearby. We only store a coarse position (rounded to about 1 km) and never share your exact whereabouts.",
    locationDenied:
      "Location permission is off. You can enable it in your device settings, then come back and try again.",
    locationUnavailable: "We could not read your position. Check that location services are on and retry.",
    photosHint: "Add up to 6 photos. The first one is your main photo."
  },
  profile: {
    deleteExplain:
      "Deleting your account permanently erases your profile, photos, likes, matches, messages and notifications. This cannot be undone.",
    safetyTips: [
      "Keep conversations on Alba until you feel comfortable.",
      "Never send money or share financial details with someone you have just met.",
      "Meet in a public place and tell a friend where you are going.",
      "Trust your instincts: block and report anyone who makes you uncomfortable."
    ]
  },
  errors: {
    generic: "Something went wrong. Please try again.",
    network: "We could not reach the server. Check your connection and try again.",
    timeout: "The request timed out. Please try again.",
    server: "Alba is having trouble right now. Please try again in a moment.",
    unauthorized: "Your session has expired. Please sign in again.",
    forbidden: "You are not allowed to do that.",
    notFound: "We could not find what you were looking for.",
    tooLarge: "That file is too large. Photos can be up to 5 MB.",
    rateLimited: "Too many attempts. Please wait a moment and try again."
  }
} as const;
