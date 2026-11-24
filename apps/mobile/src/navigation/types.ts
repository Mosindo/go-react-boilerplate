import type { NavigatorScreenParams } from "@react-navigation/native";

export type TabParamList = {
  Discover: undefined;
  Matches: undefined;
  Activity: undefined;
  Me: undefined;
};

export type RootStackParamList = {
  Welcome: undefined;
  SignIn: undefined;
  SignUp: undefined;
  ForgotPassword: { email?: string } | undefined;
  Onboarding: undefined;
  Main: NavigatorScreenParams<TabParamList> | undefined;
  ProfileDetail: { userId: string; from: "discover" | "chat"; matchId?: string; conversationId?: string };
  Chat: { conversationId: string; matchId: string; userId: string; name: string; photoUrl?: string | null };
  EditProfile: undefined;
  EditPhotos: undefined;
  Preferences: undefined;
  Privacy: undefined;
  Account: undefined;
};

declare global {
   
  namespace ReactNavigation {
    // eslint-disable-next-line @typescript-eslint/no-empty-object-type
    interface RootParamList extends RootStackParamList {}
  }
}
