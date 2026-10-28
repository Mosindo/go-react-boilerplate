import type { NavigatorScreenParams } from "@react-navigation/native";

export type AuthStackParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: { email?: string } | undefined;
  ResetPassword: { email?: string } | undefined;
};

export type TabParamList = {
  Discover: undefined;
  Chats: undefined;
  Profile: undefined;
};

export type MainStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Chat: { conversationId: string; name: string; userId: string };
  ProfileDetail: { userId: string };
  Notifications: undefined;
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Privacy: undefined;
  BlockedUsers: undefined;
  ChangePassword: undefined;
  DeleteAccount: undefined;
  Report: { userId: string; name: string; conversationId?: string };
};
