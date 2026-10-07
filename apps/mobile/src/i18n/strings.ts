/**
 * Every user-facing string lives here (English). To translate later, replace this object with a
 * locale lookup that returns the same shape. Strings with variables are small functions.
 * Screens owned by other modules may add their own sections below.
 */
export const strings = {
  app: {
    name: "Ember",
    tagline: "Meet someone worth meeting. Free, always."
  },
  common: {
    back: "Back",
    cancel: "Cancel",
    continue: "Continue",
    delete: "Delete",
    save: "Save",
    retry: "Try again",
    loading: "Loading...",
    working: "Working...",
    openSettings: "Open settings"
  },
  errors: {
    title: "Something went wrong",
    generic: "Something went wrong. Please try again.",
    network: "Can't reach the server. Check your connection and try again.",
    timeout: "The request timed out. Please try again.",
    server: "Our servers had a hiccup. Please try again in a moment.",
    rateLimited: "Too many attempts. Please wait a moment and try again.",
    noApiUrl: "The API address is not configured for this build (EXPO_PUBLIC_API_URL)."
  },
  validation: {
    emailRequired: "Enter your email address.",
    emailInvalid: "That email address doesn't look right.",
    passwordRequired: "Enter your password.",
    passwordShort: "Use at least 8 characters.",
    passwordLong: "Use at most 128 characters.",
    codeRequired: "Enter the code from your email.",
    nameRequired: "Tell us your first name.",
    nameLong: "Keep your name under 40 characters.",
    birthDateInvalid: "Enter a valid birth date.",
    underage: "You must be at least 18 to use Ember.",
    genderRequired: "Choose one option.",
    interestedInRequired: "Choose at least one option.",
    ageRangeInvalid: "Choose an age range between 18 and 99.",
    distanceInvalid: "Choose a distance between 1 and 500 km.",
    locationRequired: "Share your location or enter your city.",
    bioLong: "Keep your bio under 500 characters.",
    cityLong: "Keep the city name under 80 characters."
  },
  auth: {
    loginTitle: "Welcome back",
    registerTitle: "Create your account",
    forgotTitle: "Reset your password",
    resetTitle: "Choose a new password",
    loginAction: "Sign in",
    registerAction: "Create account",
    forgotAction: "Send reset code",
    resetAction: "Update password",
    email: "Email",
    password: "Password",
    newPassword: "New password",
    passwordHelp: "At least 8 characters.",
    resetCode: "Reset code",
    forgotIntro: "Enter your email and we'll send you a code to reset your password.",
    codeSentTitle: "Check your inbox",
    codeSentBody: "If an account exists for that email, a reset code is on its way. Enter it below.",
    haveCodeBody: "Enter the code from your email and choose a new password.",
    haveCode: "I already have a code",
    forgotLink: "Forgot your password?",
    toRegister: "New here? Create an account",
    toLogin: "Already have an account? Sign in",
    backToLogin: "Back to sign in",
    adultsOnly: "Ember is for adults. You'll confirm your birth date next.",
    resetDone: "Password updated. Sign in with your new password.",
    sessionExpired: "Your session ended. Please sign in again.",
    restoring: "Restoring your session...",
    bootErrorTitle: "Can't connect"
  },
  gender: {
    woman: "Women",
    man: "Men",
    nonbinary: "Non-binary people"
  },
  onboarding: {
    titles: {
      name: "What's your first name?",
      birthDate: "When were you born?",
      gender: "I am",
      preferences: "Who would you like to meet?",
      location: "Where are you?",
      about: "A little about you",
      photos: "Add your photos"
    },
    subtitles: {
      name: "This is what people will see on your profile.",
      birthDate: "Only your age is shown. You must be 18 or older.",
      gender: "Choose what fits you best.",
      preferences: "You can change this any time.",
      location: "We use it to show people nearby. Only an approximate area is stored.",
      about: "Write a short bio and pick up to 10 interests.",
      photos: "Add at least one photo. The first one is your main photo."
    },
    firstName: "First name",
    birthDateLabel: "Birth date",
    birthDateHelp: "Day, month, year",
    day: "DD",
    month: "MM",
    year: "YYYY",
    interestedIn: "Show me",
    minAge: "Minimum age",
    maxAge: "Maximum age",
    maxDistance: "Maximum distance",
    finish: "Start meeting people",
    photoRequired: "Add at least one photo to continue."
  },
  location: {
    use: "Use my location",
    update: "Update my location",
    savedTitle: "Location ready",
    savedBody: "We only store an approximate position (about 1 km).",
    problemTitle: "We couldn't get your location",
    denied: "Permission was denied. You can try again or type your city below.",
    blocked: "Location access is turned off for Ember. Enable it in settings, or type your city below.",
    unavailable: "Your location isn't available right now. Type your city below instead.",
    manualHelp: "Prefer not to share? Enter your city instead.",
    cityLabel: "City"
  },
  profile: {
    edit: "Edit profile",
    photos: "Photos",
    preferences: "Preferences & privacy",
    blocked: "Blocked users",
    logout: "Sign out",
    logoutTitle: "Sign out?",
    logoutBody: "You'll need to sign in again to use Ember.",
    deleteAccount: "Delete account",
    noPhoto: "No photo yet",
    noBio: "You haven't written a bio yet.",
    bio: "About me",
    bioPlaceholder: "What makes a good day for you?",
    bioCount: (used: number, max: number) => `${used}/${max}`,
    gender: "I am",
    location: "Location",
    interests: "Interests",
    interestsCount: (used: number, max: number) => `${used} of ${max} selected`,
    saved: "Saved."
  },
  photos: {
    intro: "Tap a photo to make it your main photo, reorder, replace or delete it.",
    add: "Add a photo",
    addFirst: "Add your first photo",
    main: "Main",
    makeMain: "Make main",
    earlier: "Move earlier",
    later: "Move later",
    replace: "Replace",
    tapToManage: "Tap a photo to manage it.",
    keepOne: "You need at least one photo on your profile.",
    deleteTitle: "Delete this photo?",
    deleteBody: "This can't be undone.",
    unsupported: "Please choose a JPEG or PNG image.",
    tooLarge: "That image is larger than 5 MB. Choose a smaller one.",
    permissionTitle: "Photo access needed",
    permissionBody: "Allow access to your photos in settings to add pictures to your profile.",
    photoN: (n: number) => `Photo ${n}`,
    hint: (used: number, max: number) => `${used} of ${max} photos. JPEG or PNG, up to 5 MB each.`
  },
  preferences: {
    whoTitle: "Who I want to meet",
    privacyTitle: "Privacy",
    discoverable: "Show my profile",
    discoverableHelp: "Turn off to pause your profile. Nobody new will see you.",
    showDistance: "Show my distance",
    showDistanceHelp: "When off, other people never see how far away you are."
  },
  blocked: {
    emptyTitle: "No blocked users",
    emptyBody: "People you block will appear here.",
    unblock: "Unblock",
    unblockTitle: (name: string) => `Unblock ${name}?`,
    unblockBody: "They may show up in discovery again. Previous matches are not restored.",
    unblocked: (name: string) => `${name} was unblocked.`,
    age: (age: number) => `${age} years old`
  },
  deleteAccount: {
    warningTitle: "This is permanent",
    warningBody: "Your profile, photos, matches and messages will be deleted and cannot be recovered.",
    passwordLabel: "Confirm with your password",
    action: "Delete my account",
    confirmTitle: "Delete your account?",
    confirmBody: "This cannot be undone.",
    confirmAction: "Delete forever"
  },
  tabs: {
    discover: "Discover",
    matches: "Matches",
    notifications: "Activity",
    profile: "Profile"
  },
  notifications: {
    read: "Read",
    unread: "New"
  }
} as const;
