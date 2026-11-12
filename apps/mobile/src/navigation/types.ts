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
  Notifications: undefined;
  Me: undefined;
};

export type AppStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Chat: { userId: string; firstName: string };
  ProfileView: { userId: string; fromChat?: boolean };
  EditProfile: undefined;
  EditPhotos: undefined;
  EditPreferences: undefined;
  EditInterests: undefined;
  Privacy: undefined;
  Blocked: undefined;
  Account: undefined;
};
