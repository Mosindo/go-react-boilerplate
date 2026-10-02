import type { NavigatorScreenParams } from "@react-navigation/native";

export type AuthStackParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: undefined;
  ResetPassword: { email: string };
};

export type OnboardingParamList = {
  ProfileSetup: undefined;
  PreferencesSetup: undefined;
  PhotosSetup: undefined;
};

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Alerts: undefined;
  Me: undefined;
};

export type MainStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Chat: { matchId: string; name: string; userId: string };
  ProfileDetail: { userId: string };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Settings: undefined;
  Blocked: undefined;
  ChangePassword: undefined;
  DeleteAccount: undefined;
};
