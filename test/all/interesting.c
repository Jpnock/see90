int f(int x, int y)
{
    int i_to = 10;
    int *to = &i_to;
    int count = 10;
    int n = (count + 7) / 8;
    switch (count % 8)
    {
    case 0:
        do
        {
            *to += 1;
        case 7:
            *to += 1;
        case 6:
            *to += 1;
        case 5:
            *to += 1;
        case 4:
            *to += 1;
        case 3:
            *to += 1;
        case 2:
            *to += 1;
        case 1:
            *to += 1;
        } while (--n > 0);
    }
    return i_to;

    /*
        int *my_func(x,y )

        int *a = malloc(sizeof(int) * 10);
        a[2] = 5;

        *(a+2) = 5;

        my_func(1, 3)[5] = 5;


        int c = 5;
        int *b = &c;
        int **a = &b;
        *a[5] = 6;
    */
}
