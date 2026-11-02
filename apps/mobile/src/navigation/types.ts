export type AuthStackParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  Forgot: undefined;
};

export type MainStackParamList = {
  Tabs: undefined;
  Chat: { matchId: string; partnerId: string; partnerName: string };
  UserProfile: { userId: string; matchId?: string };
  EditProfile: undefined;
  Preferences: undefined;
  Photos: undefined;
  Privacy: undefined;
  Blocked: undefined;
};

export type TabParamList = {
  Discover: undefined;
  Matches: undefined;
  Notifications: undefined;
  Me: undefined;
};
