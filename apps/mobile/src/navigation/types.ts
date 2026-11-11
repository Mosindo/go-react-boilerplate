import type { NavigatorScreenParams } from "@react-navigation/native";
import type { Card } from "../api/types";

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
  Tabs: NavigatorScreenParams<TabParamList>;
  ProfileDetail: { userId: string; card?: Card; matchId?: string };
  Chat: { matchId: string; user: Card };
  Report: { userId: string; firstName: string };
  EditProfile: undefined;
  EditPhotos: undefined;
  EditPreferences: undefined;
  EditInterests: undefined;
  Privacy: undefined;
  Account: undefined;
};
