import type { NavigatorScreenParams } from "@react-navigation/native";

export type AuthStackParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: { email?: string } | undefined;
};

export type OnboardingStackParamList = {
  Basics: undefined;
  About: undefined;
  Preferences: undefined;
  Photos: undefined;
  Location: undefined;
};

export type MainTabParamList = {
  Discover: undefined;
  Matches: undefined;
  Notifications: undefined;
  Profile: undefined;
};

export type ChatParams = {
  conversationId: string;
  userId: string;
  firstName: string;
  thumbUrl: string | null;
  /** Needed to unmatch; resolved from the caches when the entry point does not know it. */
  matchId?: string;
};

export type MainStackParamList = {
  Tabs: NavigatorScreenParams<MainTabParamList> | undefined;
  Chat: ChatParams;
  UserProfile: { userId: string };
  EditProfile: undefined;
  EditPreferences: undefined;
  Privacy: undefined;
  BlockedUsers: undefined;
  ChangePassword: undefined;
  DeleteAccount: undefined;
};
