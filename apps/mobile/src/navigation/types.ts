import type { ProfileCard } from "../api/types";

export type RootParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: undefined;
  Onboarding: undefined;
  Main: undefined;
  Chat: { conversationId: string; matchId: string; peer: ProfileCard };
  UserProfile: { userId: string; card?: ProfileCard; canSwipe?: boolean };
  Notifications: undefined;
  EditProfile: undefined;
  Preferences: undefined;
  Privacy: undefined;
  Security: undefined;
};

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Me: undefined;
};
