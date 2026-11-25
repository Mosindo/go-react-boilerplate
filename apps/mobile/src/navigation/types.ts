import type { NavigatorScreenParams } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";

export type AuthStackParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: undefined;
  ResetPassword: { email?: string } | undefined;
};

export type TabParamList = {
  Discover: undefined;
  Matches: undefined;
  Activity: undefined;
  Me: undefined;
};

export type AppStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Chat: { conversationId: string };
  ProfileDetail: { userId: string; matchId?: string };
  EditProfile: undefined;
  EditPhotos: undefined;
  Preferences: undefined;
  Privacy: undefined;
  BlockedUsers: undefined;
  Account: undefined;
};

export type AuthScreenProps<T extends keyof AuthStackParamList> = NativeStackScreenProps<AuthStackParamList, T>;
export type AppScreenProps<T extends keyof AppStackParamList> = NativeStackScreenProps<AppStackParamList, T>;
