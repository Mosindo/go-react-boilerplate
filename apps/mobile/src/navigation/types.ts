import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { Peer } from "../api/types";

export type AuthStackParamList = {
  Login: undefined;
  Register: undefined;
  ForgotPassword: { email?: string } | undefined;
};

export type MainStackParamList = {
  Tabs: undefined;
  Chat: { conversationId: string; peer: Peer; matchId?: string };
  UserProfile: { userId: string; conversationId?: string; matchId?: string };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Settings: undefined;
  Blocked: undefined;
  ChangePassword: undefined;
  DeleteAccount: undefined;
  Report: { userId: string; name: string };
};

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Activity: undefined;
  Me: undefined;
};

export type AuthScreenProps<T extends keyof AuthStackParamList> = NativeStackScreenProps<
  AuthStackParamList,
  T
>;
export type MainScreenProps<T extends keyof MainStackParamList> = NativeStackScreenProps<
  MainStackParamList,
  T
>;
