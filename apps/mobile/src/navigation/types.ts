import type { NavigatorScreenParams } from "@react-navigation/native";

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Notifications: undefined;
  Profile: undefined;
};

export type RootStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Chat: { conversationId: string; name?: string };
  ProfileDetail: { userId: string; canSwipe?: boolean };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Privacy: undefined;
  ChangePassword: undefined;
};

export type AuthStackParamList = {
  Login: undefined;
  Forgot: { email?: string } | undefined;
};
