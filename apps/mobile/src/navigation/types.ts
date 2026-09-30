import type { NavigatorScreenParams } from "@react-navigation/native";

export type AuthStackParamList = {
  Welcome: undefined;
  SignIn: undefined;
  SignUp: undefined;
  ForgotPassword: { email?: string } | undefined;
};

export type MainTabParamList = {
  Discover: undefined;
  Messages: undefined;
  Notifications: undefined;
  Me: undefined;
};

export type AppStackParamList = {
  Tabs: NavigatorScreenParams<MainTabParamList>;
  ProfileDetail: { userId: string; fromDiscovery?: boolean };
  Chat: { conversationId: string; name?: string };
  EditProfile: undefined;
  Preferences: undefined;
  Settings: undefined;
  BlockedUsers: undefined;
  Moderation: undefined;
};
