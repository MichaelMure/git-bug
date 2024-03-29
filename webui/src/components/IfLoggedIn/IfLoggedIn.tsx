import * as React from 'react';

import { useCurrentIdentityQuery } from '../Identity/CurrentIdentity.generated';

type Props = { children: () => React.ReactNode };
const IfLoggedIn = ({ children }: Props) => {
  const { loading, error, data } = useCurrentIdentityQuery();

  if (error || loading || !data?.repository?.userIdentity) return null;

  return <>{children()}</>;
};

export default IfLoggedIn;
