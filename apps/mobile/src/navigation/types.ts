import type { NativeStackScreenProps } from "@react-navigation/native-stack";

export type AuthStackParamList = {
  SignIn: undefined;
  ForgotPassword: { email?: string } | undefined;
};

export type MainTabParamList = {
  Discover: undefined;
  Matches: undefined;
  Activity: undefined;
  Me: undefined;
};

export type MainStackParamList = {
  Tabs: undefined;
  Chat: { matchId: string; name: string };
  ProfileDetail: { userId: string; matchId?: string };
  EditProfile: undefined;
  Preferences: undefined;
  Privacy: undefined;
  Account: undefined;
};

export type MainScreenProps<T extends keyof MainStackParamList> = NativeStackScreenProps<MainStackParamList, T>;
export type AuthScreenProps<T extends keyof AuthStackParamList> = NativeStackScreenProps<AuthStackParamList, T>;
