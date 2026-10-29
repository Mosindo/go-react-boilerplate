import type { NavigatorScreenParams } from "@react-navigation/native";
import type { MatchUser } from "../api/types";

export type MainTabParamList = {
  Discover: undefined;
  Chats: undefined;
  Notifications: undefined;
  Me: undefined;
};

export type RootStackParamList = {
  // Signed out
  Welcome: undefined;
  SignIn: undefined;
  SignUp: undefined;
  ForgotPassword: undefined;
  ResetPassword: { email: string };
  // Signed in, onboarding unfinished
  Onboarding: undefined;
  // Signed in
  Main: NavigatorScreenParams<MainTabParamList>;
  Chat: { conversationId: string; user: MatchUser };
  ProfileDetail: { userId: string; firstName?: string; allowActions?: boolean };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Settings: undefined;
  Blocked: undefined;
  ChangePassword: undefined;
};
