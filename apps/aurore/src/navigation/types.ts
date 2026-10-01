import type { NavigatorScreenParams } from "@react-navigation/native";

export type TabParamList = {
  Discover: undefined;
  Matches: undefined;
  Notifications: undefined;
  Me: undefined;
};

export type RootStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList>;
  Chat: { conversationId: string; userId: string; name: string };
  ProfileDetail: { userId: string; matchId?: string; conversationId?: string };
  Report: { userId: string; name: string };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Settings: undefined;
  Blocked: undefined;
};

export type AuthStackParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  Forgot: { email?: string } | undefined;
};
