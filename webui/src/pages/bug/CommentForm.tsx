import Button from '@mui/material/Button';
import Paper from '@mui/material/Paper';
import { Theme } from '@mui/material/styles';
import makeStyles from '@mui/styles/makeStyles';
import * as React from 'react';
import { useState, useRef } from 'react';

import CommentInput from '../../components/CommentInput/CommentInput';
import CloseBugButton from 'src/components/CloseBugButton';
import CloseBugWithCommentButton from 'src/components/CloseBugWithCommentButton';
import ReopenBugButton from 'src/components/ReopenBugButton';
import ReopenBugWithCommentButton from 'src/components/ReopenBugWithCommentButton';

import { BugFragment } from './Bug.generated';
import { useAddCommentMutation } from './CommentForm.generated';
import { TimelineDocument } from './TimelineQuery.generated';

type StyleProps = { loading: boolean };
const useStyles = makeStyles<Theme, StyleProps>((theme) => ({
  container: {
    padding: theme.spacing(0, 2, 2, 2),
  },
  actions: {
    display: 'flex',
    gap: '1em',
    justifyContent: 'flex-end',
  },
  greenButton: {
    marginLeft: '8px',
    backgroundColor: theme.palette.success.main,
    color: theme.palette.success.contrastText,
    '&:hover': {
      backgroundColor: theme.palette.success.dark,
      color: theme.palette.primary.contrastText,
    },
  },
}));

type Props = {
  bug: BugFragment;
};

function CommentForm({ bug }: Props) {
  const [addComment, { loading }] = useAddCommentMutation();
  const [issueComment, setIssueComment] = useState('');
  const [inputProp, setInputProp] = useState<any>('');
  const classes = useStyles({ loading });
  const form = useRef<HTMLFormElement>(null);

  const submit = () => {
    addComment({
      variables: {
        input: {
          prefix: bug.id,
          message: issueComment,
        },
      },
      refetchQueries: [
        // TODO: update the cache instead of refetching
        {
          query: TimelineDocument,
          variables: {
            id: bug.id,
            first: 100,
          },
        },
      ],
      awaitRefetchQueries: true,
    }).then(() => resetForm());
  };

  function resetForm() {
    setInputProp({
      value: '',
    });
  }

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (issueComment.length > 0) submit();
  };

  function getBugStatusButton() {
    if (bug.status === 'OPEN' && issueComment.length > 0) {
      return (
        <CloseBugWithCommentButton
          bug={bug}
          comment={issueComment}
          postClick={resetForm}
        />
      );
    }
    if (bug.status === 'OPEN') {
      return <CloseBugButton bug={bug} />;
    }
    if (bug.status === 'CLOSED' && issueComment.length > 0) {
      return (
        <ReopenBugWithCommentButton
          bug={bug}
          comment={issueComment}
          postClick={resetForm}
        />
      );
    }
    return <ReopenBugButton bug={bug} />;
  }

  return (
    <Paper className={classes.container}>
      <form onSubmit={handleSubmit} ref={form}>
        <CommentInput
          inputProps={inputProp}
          loading={loading}
          onChange={(comment: string) => setIssueComment(comment)}
        />
        <div className={classes.actions}>
          {getBugStatusButton()}
          <Button
            className={classes.greenButton}
            variant="contained"
            color="primary"
            type="submit"
            disabled={loading || issueComment.length === 0}
          >
            Comment
          </Button>
        </div>
      </form>
    </Paper>
  );
}

export default CommentForm;
