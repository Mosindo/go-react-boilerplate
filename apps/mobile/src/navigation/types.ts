import type { NavigatorScreenParams } from "@react-navigation/native";

export type AuthStackParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: { email?: string } | undefined;
};

export type TabParamList = {
  Discover: undefined;
  Matches: undefined;
  Activity: undefined;
  Me: undefined;
};

export type AppStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Chat: { conversationId: string; matchId: string; userId: string; firstName: string };
  ProfileDetail: { userId: string; fromDiscover?: boolean };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Settings: undefined;
  BlockedUsers: undefined;
  ChangePassword: undefined;
  DeleteAccount: undefined;
  Report: { userId: string; firstName: string };
};
