import type { NavigatorScreenParams } from "@react-navigation/native";

export type AuthStackParamList = {
  Login: undefined;
  Register: undefined;
  ForgotPassword: { email?: string } | undefined;
};

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Notifications: undefined;
  Me: undefined;
};

export type MainStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Chat: { conversationId: string };
  ProfileDetail: { id: string; swipeable?: boolean };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Privacy: undefined;
};
